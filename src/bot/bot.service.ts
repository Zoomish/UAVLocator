import { forwardRef, Inject, Injectable, OnModuleInit } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import { InjectRepository } from '@nestjs/typeorm'
import TelegramBot, * as telegram from 'node-telegram-bot-api'
import { Repository } from 'typeorm'
import { UpdateBotDto } from './dto/update-bot.dto'
import { Bot } from './entities/bot.entity'
import {  HandleService } from './services'

@Injectable()
export class BotService implements OnModuleInit {
    constructor(
        @Inject(forwardRef(() => HandleService))
        private readonly handleService: HandleService,
        private readonly configService: ConfigService,
        @InjectRepository(Bot)
        private readonly botRepository: Repository<Bot>
    ) {}

    async onModuleInit() {
        const telegramToken = this.configService.get('TELEGRAM_TOKEN')
        const bot: TelegramBot = new telegram(telegramToken, {
            polling: true,
            onlyFirstMatch: true,
        })
        await this.initBot(bot)
    }

    async initBot(bot: TelegramBot) {
        global.bot = bot
        await bot.setMyCommands([
            {
                command: '/start',
                description: 'Начальное меню',
            },
            {
                command: '/settings',
                description: '⚙️ Настройки',
            },
        ])
        bot.on('message', async (msg: TelegramBot.Message) => {
            return await this.handleService.handleMessage(msg)
        })
    }
    async update(tgId: number, dto: UpdateBotDto) {
        const filter = await this.botRepository.findOne({
            where: { user: { tgId } },
        })
        return await this.botRepository.save(Object.assign(filter, dto))
    }

    async findOne(tgId: number) {
        const bot = await this.botRepository.findOne({
            where: { user: { tgId } },
        })
        return bot
    }
}
