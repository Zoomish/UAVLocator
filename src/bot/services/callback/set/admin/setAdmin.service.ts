import { forwardRef, Inject, Injectable, Logger } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'
import { UserService } from 'src/user/user.service'
import { GetAdminCallbackService } from '../../get'

@Injectable()
export class SetAdminCallbackService {
    constructor(
        @Inject(forwardRef(() => UserService))
        private readonly userService: UserService,
        private readonly getAdminCallbackService: GetAdminCallbackService
    ) {}
    private readonly logger = new Logger(SetAdminCallbackService.name)

    async handleSetAdmin(
        text: string | undefined,
        callbackQuery: TelegramBot.CallbackQuery
    ) {
        const bot: TelegramBot = global.bot
        const texts = text.split('=')

        switch (texts[0]) {
            case 'admin': {
                let user = await this.userService.findOne(+texts[1])
                bot.answerCallbackQuery(callbackQuery.id, {
                    text: `Вы ${user.admin ? 'убираете' : 'делаете'} пользователя админом`,
                }).catch((error) =>
                    this.logger.error('Error answering callback: ' + error)
                )
                user = await this.userService.update(+texts[1], {
                    admin: !user.admin,
                })
                await this.getAdminCallbackService.sendUserInfo(
                    user,
                    callbackQuery.message.message_id
                )
                break
            }
        }
    }
}
