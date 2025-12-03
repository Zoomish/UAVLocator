import { Injectable, Logger } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import TelegramBot from 'node-telegram-bot-api'
import { BotService } from 'src/bot/bot.service'
import { UserService } from 'src/user/user.service'

@Injectable()
export class StartService {
    private readonly logger = new Logger(StartService.name)

    constructor(
        private readonly userService: UserService,
        private readonly configService: ConfigService
    ) {}

    async startMessage() {
        const bot: TelegramBot = global.bot
        const msg: TelegramBot.Message = global.msg
        let user = await this.userService.findOne(msg.chat.id)

        if (user) {
            user = await this.userService.update(msg.chat.id, {
                username: msg.chat.username,
                name: msg.chat.first_name,
            })
        }

        await bot
            .deleteMessage(msg.chat.id, msg.message_id)
            .catch((error) =>
                this.logger.error('Error deleting start message: ' + error)
            )
    }
}
