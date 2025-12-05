import { Injectable, Logger } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'

@Injectable()
export class SendInfoService {
    private readonly logger = new Logger(SendInfoService.name)

    async sendInfo(tgId: number, message: string) {
        const bot: TelegramBot = global.bot
        await bot.sendMessage(tgId, message).catch((e) => this.logger.error(e))
    }
}
