import { Injectable, Logger } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import { InjectRepository } from '@nestjs/typeorm'
import { Bot } from 'src/bot/entities/bot.entity'
import { NoSessionService, SendInfoService } from 'src/bot/services'
import { TelegramClient } from 'telegram'
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
            },
        })
    }

    private normalizeText(text: string): string {
        return text
            .toLowerCase()
            .replace(/[ёе]/g, 'е')
            .replace(/[йи]/g, 'и')
            .replace(/[ъь]/g, '')
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
        this.logger.log(`📡 Мониторинг канала: ${channelUsername}`)

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
            }
        })

        console.log('✅ Мониторинг запущен. Жду новые посты...')

        await client.connect()
    }
}
