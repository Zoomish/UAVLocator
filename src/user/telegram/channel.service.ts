import { Injectable, Logger } from '@nestjs/common'
import { Api } from 'telegram'
import { TelegramClientService } from './telegramClient.service'

interface ProcessedMessage {
    id: number
    text: string
    originalMessage: any
}

@Injectable()
export class ChannelService {
    private readonly logger = new Logger(ChannelService.name)

    constructor(private telegramClientService: TelegramClientService) {}

    async getChannelEntity(channelUsername: string) {
        this.logger.log(`Получение канала: ${channelUsername}`)
        const client = await this.telegramClientService.getClient()
        const channel = await client.getEntity(channelUsername)
        return channel
    }

    async getChannelDialogs() {
        const client = await this.telegramClientService.getClient()
        return await client.getDialogs({})
    }

    async getChannelFullInfo(channel: any) {
        const client = await this.telegramClientService.getClient()
        const fullChannel = (await client.invoke(
            new Api.channels.GetFullChannel({ channel })
        )) as Api.messages.ChatFull

        return fullChannel
    }

    async getChannelHistory(channel: any, limit: number) {
        const client = await this.telegramClientService.getClient()
        const result = (await client.invoke(
            new Api.messages.GetHistory({
                peer: channel,
                limit: limit,
                offsetId: 0,
                offsetDate: 0,
                addOffset: 0,
                maxId: 0,
                minId: 0,
            })
        )) as Api.messages.Messages

        return result.messages || []
    }

    async getReplyMessage(channel: any, messageId: number) {
        const client = await this.telegramClientService.getClient()
        const reply = (await client.invoke(
            new Api.channels.GetMessages({
                channel: channel,
                id: [new Api.InputMessageID({ id: messageId })],
            })
        )) as Api.messages.Messages

        return reply?.messages?.[0] || null
    }

    async markMessagesAsRead(channel: any, maxId: number) {
        const client = await this.telegramClientService.getClient()
        await client.invoke(
            new Api.channels.ReadHistory({
                channel: channel,
                maxId: maxId,
            })
        )
        this.logger.log(`Сообщения до ${maxId} отмечены как прочитанные`)
    }

    private isMessageWithText(
        message: any
    ): message is { message: string; id: number } {
        return (
            message &&
            typeof message === 'object' &&
            'message' in message &&
            typeof message.message === 'string' &&
            'id' in message
        )
    }

    private getMessageText(message: any): string {
        if (this.isMessageWithText(message)) {
            return message.message.replace('📡Локатор России - @locatorru', '')
        }
        return ''
    }

    async getUnreadMessagesData(channelUsername: string): Promise<{
        unreadMessages: ProcessedMessage[]
        channel: any
        maxId: number
    }> {
        this.logger.log(
            `Проверка непрочитанных сообщений в канале: ${channelUsername}`
        )

        const channel = await this.getChannelEntity(channelUsername)
        const dialogs = await this.getChannelDialogs()

        const channelDialog = dialogs.find((d) =>
            d.entity.id.equals(channel.id)
        )
        if (!channelDialog) {
            throw new Error('Канал не найден в диалогах')
        }

        if (channelDialog.unreadCount === 0) {
            this.logger.log('Непрочитанных сообщений нет')
            return { unreadMessages: [], channel: null, maxId: 0 }
        }

        this.logger.log(`Непрочитано сообщений: ${channelDialog.unreadCount}`)

        const fullChannel = await this.getChannelFullInfo(channel)
        const messages = await this.getChannelHistory(
            channel,
            Math.min(channelDialog.unreadCount + 10, 100)
        )

        const chatFull = fullChannel.fullChat as Api.ChannelFull
        const readInboxMaxId = chatFull.readInboxMaxId || 0

        const unreadMessages = messages.filter(
            (msg: any) => msg.id > readInboxMaxId
        )

        const processedMessages: ProcessedMessage[] = []

        for (const message of unreadMessages) {
            if (!this.isMessageWithText(message)) {
                continue
            }

            let text = ''

            if (message?.replyTo?.replyToMsgId) {
                try {
                    const replyMessage = await this.getReplyMessage(
                        channel,
                        message.replyTo.replyToMsgId
                    )
                    if (replyMessage && this.isMessageWithText(replyMessage)) {
                        const replyText = this.getMessageText(replyMessage)
                        if (replyText) {
                            text += `<blockquote expandable>${replyText}</blockquote>`
                        }
                    }
                } catch (error) {
                    // Игнорируем ошибки получения ответных сообщений
                }
            }

            text += this.getMessageText(message)

            processedMessages.push({
                id: message.id,
                text: text,
                originalMessage: message,
            })
        }

        const maxId =
            processedMessages.length > 0
                ? Math.max(...processedMessages.map((msg) => msg.id))
                : 0

        this.logger.log(
            `Обработано ${processedMessages.length} непрочитанных сообщений`
        )

        return {
            unreadMessages: processedMessages,
            channel: channel,
            maxId: maxId,
        }
    }
}
