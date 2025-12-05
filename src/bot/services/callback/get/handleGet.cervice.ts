import { forwardRef, Inject, Injectable } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'
import { SettingsService } from '../../handle'
import { GetAdminCallbackService } from './admin'

@Injectable()
export class HandleGetService {
    constructor(
        @Inject(forwardRef(() => SettingsService))
        private readonly settingsService: SettingsService,
        private readonly getAdminCallbackService: GetAdminCallbackService
    ) {}

    async handleGet(text: string, callbackQuery: TelegramBot.CallbackQuery) {
        const bot: TelegramBot = global.bot
        const texts = text.split('-')

        switch (texts[0]) {
            case 'settings': {
                bot.answerCallbackQuery(callbackQuery.id, {
                    text: 'Перейти в настройки',
                })
                return await this.settingsService.settings(
                    callbackQuery.message.message_id
                )
            }
            case 'admin': {
                return await this.getAdminCallbackService.handleGetAdmin(
                    texts[1],
                    callbackQuery
                )
            }
        }
    }
}
