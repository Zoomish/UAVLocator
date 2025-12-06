import { forwardRef, Inject, Injectable, Logger } from '@nestjs/common'
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
    private readonly logger = new Logger(HandleGetService.name)

    async handleGet(text: string, callbackQuery: TelegramBot.CallbackQuery) {
        const bot: TelegramBot = global.bot
        const texts = text.split('-')

        switch (texts[0]) {
            case 'settings': {
                bot.answerCallbackQuery(callbackQuery.id, {
                    text: 'Перейти в настройки',
                }).catch((error) =>
                    this.logger.error('Error answering callback: ' + error)
                )
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
