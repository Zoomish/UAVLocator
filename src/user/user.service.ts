import { Injectable, Logger } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import { InjectRepository } from '@nestjs/typeorm'
import { Bot } from 'src/bot/entities/bot.entity'
import { NoSessionService } from 'src/bot/services'
import {  TelegramClient } from 'telegram'
import { Repository } from 'typeorm'
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
        private readonly noSessionService: NoSessionService
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

    async update(tgId: number, dto: UpdateUserDto) {
        const user = await this.findOne(tgId)
        return await this.userRepository.save(Object.assign(user, dto))
    }

    async checkLocations() {
        const apiId = this.configService.get('API_ID')
        const apiHash = this.configService.get('API_HASH')
        const session = this.configService.get('SESSION')
        const client = new TelegramClient(session, apiId, apiHash, {
            connectionRetries: 5,
        })
        await client.connect()
        try {
            await client.getMe()
        } catch (error) {
            const admin = await this.findAdmin()
            if (admin) {
                this.noSessionService.NoSession(admin.tgId)
            }
        }
        this.logger.log('✅ User Connected')
        await client.disconnect()
    }
}
