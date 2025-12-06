import { forwardRef, Inject, Injectable, Logger } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'
import { BotService } from 'src/bot/bot.service'
import { UserService } from 'src/user/user.service'
import { SettingsService } from '../../handle'
import { SetAdminCallbackService } from './admin'
import { SetLocationsCallbackService } from './locations'

@Injectable()
export class HandleSetService {
    constructor(
        private readonly setAdminCallbackService: SetAdminCallbackService,
        @Inject(forwardRef(() => BotService))
        private readonly botService: BotService,
        @Inject(forwardRef(() => UserService))
        private readonly userService: UserService,
        private readonly setLocationsCallbackService: SetLocationsCallbackService,
        @Inject(forwardRef(() => SettingsService))
        private readonly settingsService: SettingsService
    ) {}

    private readonly logger = new Logger(HandleSetService.name)

    async handleSet(text: string, callbackQuery: TelegramBot.CallbackQuery) {
        const bot: TelegramBot = global.bot
        const texts = text.split('-')

        switch (texts[0]) {
            case 'admin': {
                return await this.setAdminCallbackService.handleSetAdmin(
                    texts[1],
                    callbackQuery
                )
            }
            case 'locations': {
                return await this.setLocationsCallbackService.handleSetLocations(
                    callbackQuery
                )
            }
            case 'notifications': {
                bot.answerCallbackQuery(callbackQuery.id, {
                    text: `Вы ${texts[1] === 'true' ? 'включаете' : 'выключаете'} уведомления`,
                }).catch((error) =>
                    this.logger.error('Error answering callback: ' + error)
                )
                await this.userService.update(callbackQuery.message.chat.id, {
                    notifications: texts[1] === 'true',
                })
                return await this.settingsService.settings(
                    callbackQuery.message.message_id
                )
            }
        }
    }
}
