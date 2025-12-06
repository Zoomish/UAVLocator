import { forwardRef, Inject, Injectable, Logger } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'
import { BotService } from 'src/bot/bot.service'
import { SettingsService } from '../../handle'

@Injectable()
export class HandleCancelService {
    constructor(
        @Inject(forwardRef(() => SettingsService))
        private readonly settingsService: SettingsService,
        @Inject(forwardRef(() => BotService))
        private readonly botService: BotService
    ) {}
    private readonly logger = new Logger(HandleCancelService.name)

    async handleCancel(text: string, callbackQuery: TelegramBot.CallbackQuery) {
        const bot: TelegramBot = global.bot
        const texts = text.split('-')
        const botService = await this.botService.findOne(
            callbackQuery.message.chat.id
        )

        await this.botService.update(callbackQuery.message.chat.id, {
            waitingFor: null,
            msg_id: null,
            msg_id_delete: null,
        })
        switch (texts[0]) {
            case 'error': {
                bot.answerCallbackQuery(callbackQuery.id, {
                    text: `Вы отменили отправку ошибки`,
                }).catch((error) =>
                    this.logger.error('Error answering callback: ' + error)
                )
                await bot
                    .deleteMessage(
                        callbackQuery.message.chat.id,
                        botService.msg_id
                    )
                    .catch((error) =>
                        this.logger.error('Error deleting message: ' + error)
                    )
                if (botService.msg_id_delete) {
                    await bot
                        .deleteMessage(
                            callbackQuery.message.chat.id,
                            botService.msg_id_delete
                        )
                        .catch((error) =>
                            this.logger.error(
                                'Error deleting message: ' + error
                            )
                        )
                }
                break
            }
            case 'locations': {
                bot.answerCallbackQuery(callbackQuery.id, {
                    text: `Вы отменили отправку локаций`,
                }).catch((error) =>
                    this.logger.error('Error answering callback: ' + error)
                )
                return await this.settingsService.settings(botService.msg_id)
            }
        }
    }
}
