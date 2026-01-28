import { forwardRef, Inject, Injectable, Logger } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'
import { UserService } from 'src/user/user.service'

@Injectable()
export class SendInfoService {
    constructor(
        @Inject(forwardRef(() => UserService))
        private readonly userService: UserService
    ) {}
    private readonly logger = new Logger(SendInfoService.name)

    async sendInfo(tgId: number, message: string) {
        const bot: TelegramBot = global.bot
        await bot
            .sendMessage(tgId, message, {
                parse_mode: 'HTML',
            })
            .catch((error) => {
                if (error.response && error.response.statusCode === 403) {
                    this.logger.warn(
                        `User ${tgId} blocked the bot. Disabling the bot.`
                    )
                    this.userService.update(tgId, { notifications: false })
                } else {
                    this.logger.error(error)
                }
            })
    }
}
