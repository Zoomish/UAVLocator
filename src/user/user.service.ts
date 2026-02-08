import { Injectable, Logger } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import { InjectRepository } from '@nestjs/typeorm'
import { Bot } from 'src/bot/entities/bot.entity'
import { NoSessionService, SendInfoService } from 'src/bot/services'
import { Raw, Repository } from 'typeorm'
import { CreateUserDto } from './dto/create-user.dto'
import { UpdateUserDto } from './dto/update-user.dto'
import { User } from './entities/user.entity'
import { ChannelService } from './telegram/channel.service'

@Injectable()
export class UserService {
    private readonly logger = new Logger(UserService.name)

    constructor(
        @InjectRepository(User)
        private readonly userRepository: Repository<User>,
        @InjectRepository(Bot)
        private readonly botRepository: Repository<Bot>,
        private readonly configService: ConfigService,
        private readonly noSessionService: NoSessionService,
        private readonly sendInfoService: SendInfoService,
        private readonly channelService: ChannelService
    ) {}

    async create(createUserDto: CreateUserDto) {
        const newUser = this.userRepository.create(createUserDto)
        const user = await this.userRepository.save(newUser)

        const newBot = new Bot()
        newBot.user = user
        await this.botRepository.save(newBot)

        return user
    }

    async findOne(tgId: number) {
        return await this.userRepository.findOne({
            where: { tgId },
        })
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
            .replace(/[^а-я0-9\s]/g, ' ')
            .replace(/\s+/g, ' ')
            .trim()
    }

    async update(tgId: number, dto: UpdateUserDto) {
        const user = await this.findOne(tgId)
        return await this.userRepository.save(Object.assign(user, dto))
    }

    async checkUnreadMessages() {
        const channelUsername = this.configService.get<string>('CHANNEL')

        try {
            const { unreadMessages, channel, maxId } =
                await this.channelService.getUnreadMessagesData(channelUsername)

            if (unreadMessages.length === 0) {
                return
            }

            for (const message of unreadMessages) {
                const users = await this.findAllWithLocation(message.text)

                for (const user of users) {
                    await this.sendInfoService.sendInfo(user.tgId, message.text)
                }
            }

            if (channel) {
                await this.channelService.markMessagesAsRead(channel, maxId)
            }
        } catch (error) {
            this.logger.error(`Ошибка проверки сообщений: ${error.message}`)
            const admin = await this.findAdmin()
            if (admin) {
                await this.noSessionService.NoSession(admin.tgId)
            }
        }
    }
}
