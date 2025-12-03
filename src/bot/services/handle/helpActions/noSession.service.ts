import { Injectable, Logger } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'
import { BotService } from 'src/bot/bot.service'

@Injectable()
export class NoSessionService {
    private readonly logger = new Logger(NoSessionService.name)

    async NoSession(tgId: number) {
        const bot: TelegramBot = global.bot
        await bot.sendMessage(tgId, 'No session').catch((e) => this.logger.error(e))
    }
}
