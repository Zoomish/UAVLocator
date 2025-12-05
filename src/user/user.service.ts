import { Injectable, Logger } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import { InjectRepository } from '@nestjs/typeorm'
import { Bot } from 'src/bot/entities/bot.entity'
import { NoSessionService, SendInfoService } from 'src/bot/services'
import { Api, TelegramClient } from 'telegram'
import { StringSession } from 'telegram/sessions'
import { Raw, Repository } from 'typeorm'
import { CreateUserDto } from './dto/create-user.dto'
import { UpdateUserDto } from './dto/update-user.dto'
import { User } from './entities/user.entity'

@Injectable()
export class UserService {
    constructor(
        @InjectRepository(User)
        private readonly userRepository: Repository<User>,
        @InjectRepository(Bot)
        private readonly botRepository: Repository<Bot>,
        private readonly configService: ConfigService,
        private readonly noSessionService: NoSessionService,
        private readonly sendInfoService: SendInfoService
    ) {}
    private readonly logger = new Logger(UserService.name)

    async create(createUserDto: CreateUserDto) {
        const newUser = this.userRepository.create(createUserDto)
        const user = await this.userRepository.save(newUser)

        const newBot = new Bot()
        newBot.user = user
        await this.botRepository.save(newBot)

        return user
    }

    async findOne(tgId: number) {
        const user = await this.userRepository.findOne({
            where: { tgId },
        })
        return user
    }

    async findAdmin() {
        return await this.userRepository.findOne({
            where: { admin: true },
        })
    }

    async findAll() {
        return await this.userRepository.find()
    }

    async findAllWithLocation(message: string) {
        const normalizedMessage = this.normalizeText(message)

        return this.userRepository.find({
            where: {
                locations: Raw(
                    (alias) =>
                        `EXISTS (
                    SELECT 1 
                    FROM unnest(${alias}) AS pattern_item 
                    WHERE position(pattern_item in :message) > 0
                )`,
                    { message: normalizedMessage }
                ),
                notifications: true,
            },
        })
    }

    private normalizeText(text: string): string {
        return text
            .toLowerCase()
            .replace(/[ёе]/g, 'е')
            .replace(/[йи]/g, 'и')
            .replace(/[^а-я0-9\s]/g, ' ')
            .replace(/\s+/g, ' ')
            .trim()
    }

    async update(tgId: number, dto: UpdateUserDto) {
        const user = await this.findOne(tgId)
        return await this.userRepository.save(Object.assign(user, dto))
    }

    async monitorCommentsInRealTime() {
        const apiId = this.configService.get('API_ID')
        const apiHash = this.configService.get('API_HASH')
        const session = this.configService.get('SESSION')
        const channelUsername = this.configService.get('CHANNEL')
        const client = new TelegramClient(
            new StringSession(session),
            +apiId,
            apiHash,
            {
                connectionRetries: 5,
            }
        )
        await client.connect().catch(async (e) => {
            this.logger.error(e)
            const admin = await this.findAdmin()
            if (admin) {
                this.noSessionService.NoSession(admin.tgId)
            }
        })

        const channel = await client.getEntity(channelUsername)
        this.logger.log(`📡 Monitoring Channel: ${channelUsername}`)

        client.addEventHandler(async (event) => {
            const message = event.message
            if (message && channel.id.equals(message?.peerId?.channelId)) {
                if (!message.replyTo) {
                    const text = message.message.replace(
                        '📡Локатор России - @locatorru',
                        ''
                    )
                    const users = await this.findAllWithLocation(text)
                    for (const user of users) {
                        this.sendInfoService.sendInfo(user.tgId, text)
                    }
                }
                await client.invoke(
                    new Api.channels.ReadHistory({
                        channel: channel,
                        maxId: message.id,
                    })
                )
            }
        })

        this.logger.log('✅ Monitoring started')

        await client.connect()
    }

    async checkUnreadMessages() {
        const apiId = this.configService.get('API_ID')
        const apiHash = this.configService.get('API_HASH')
        const session = this.configService.get('SESSION')
        const channelUsername = this.configService.get('CHANNEL')
        const client = new TelegramClient(
            new StringSession(session),
            +apiId,
            apiHash,
            {
                connectionRetries: 5,
            }
        )
        await client.connect().catch(async (e) => {
            this.logger.error(e)
            const admin = await this.findAdmin()
            if (admin) {
                this.noSessionService.NoSession(admin.tgId)
            }
        })
        const channel = await client.getEntity(channelUsername)

        const dialogs = await client.getDialogs({})
        const channelDialog = dialogs.find((d) =>
            d.entity.id.equals(channel.id)
        )

        if (!channelDialog) {
            this.logger.error('Канал не найден в диалогах')
            return []
        }
        if (channelDialog.unreadCount > 0) {
            const fullChannel = await client.invoke(
                new Api.channels.GetFullChannel({
                    channel: channel,
                })
            )
            const result = await client.invoke(
                new Api.messages.GetHistory({
                    peer: channel,
                    limit: Math.min(channelDialog.unreadCount + 10, 100),
                    offsetId: 0,
                    offsetDate: 0,
                    addOffset: 0,
                    maxId: 0,
                    minId: 0,
                })
            )

            // @ts-ignore
            const messages = result.messages

            const unreadMessages = messages.filter(
                // @ts-ignore
                (msg) => msg.id > fullChannel.fullChat.readInboxMaxId
            )
            for (const message of unreadMessages) {
                const text = message.message.replace(
                    '📡Локатор России - @locatorru',
                    ''
                )
                const users = await this.findAllWithLocation(text)
                for (const user of users) {
                    this.sendInfoService.sendInfo(user.tgId, text)
                }

                await client.invoke(
                    new Api.channels.ReadHistory({
                        channel: channel,
                        maxId: message.id,
                    })
                )
            }

            this.logger.log(
                `✅ Checked ${unreadMessages.length} unread messages`
            )

            await client.disconnect()
        }
    }
}
