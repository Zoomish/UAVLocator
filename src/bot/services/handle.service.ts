import { forwardRef, Inject, Injectable, Logger } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import TelegramBot from 'node-telegram-bot-api'
import { BotService } from '../bot.service'
import { NoCommandsService } from './handle/helpActions'
import { InputTextService } from './handle/inputText.service'
import { SettingsService } from './handle/settings.service'
import { StartService } from './handle/start.service'

@Injectable()
export class HandleService {
    constructor(
        @Inject(forwardRef(() => BotService))
        private readonly botService: BotService,
        private readonly startService: StartService,
        private readonly inputTextService: InputTextService,
        private readonly settingsService: SettingsService,
        private readonly noCommandsService: NoCommandsService,
        private readonly configService: ConfigService
    ) {}
    private readonly logger = new Logger(HandleService.name)

    async handleMessage(msg: TelegramBot.Message) {
        if (msg.chat.type !== 'private') return
        global.msg = msg
        return this.processTextMessage(msg.text, msg)
    }

    private async processTextMessage(text: string, msg: TelegramBot.Message) {
        const bot: TelegramBot = global.bot
        const chatId = msg.chat.id
        await bot.sendChatAction(chatId, 'typing')
        const botService = await this.botService.findOne(chatId)
        if (botService?.msg_id) {
            if (botService.waitingFor === 'apply') {
                await bot
                    .deleteMessage(chatId, msg.message_id)
                    .catch((error) =>
                        this.logger.error('Error deleting message: ' + error)
                    )
                if (botService.msg_id_delete) {
                    return
                } else {
                    const message = await bot
                        .sendMessage(
                            chatId,
                            '⏳ <b>Пожалуйста, подождите...</b> Сейчас идёт процесс отправки откликов. Это может занять несколько минут, поэтому просьба не прерывать работу бота и немного подождать. Спасибо за понимание! 🙏',
                            {
                                parse_mode: 'HTML',
                            }
                        )
                        .catch((error) =>
                            this.logger.error(
                                'Error sending wait message: ' + error
                            )
                        )
                    if (message) {
                        return await this.botService.update(chatId, {
                            msg_id_delete: message.message_id,
                        })
                    }
                    return
                }
            } else if (msg.entities?.length) {
                if (msg.entities[0].type === 'bot_command') {
                    return await this.noCommandsService.NoCommands()
                }
            }
            return await this.inputTextService.handleInputText(text)
        }

        switch (text) {
            case '/start':
                return this.startService.startMessage()
            case '/settings':
                return this.settingsService.settings()
            default:
                break
        }

        await bot.deleteMessage(chatId, msg.message_id)
    }
}
