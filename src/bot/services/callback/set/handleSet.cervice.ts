import { forwardRef, Inject, Injectable, Logger } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'
import { BotService } from 'src/bot/bot.service'
import { SetAdminCallbackService } from './admin'

@Injectable()
export class HandleSetService {
    constructor(
        private readonly setAdminCallbackService: SetAdminCallbackService,
        @Inject(forwardRef(() => BotService))
        private readonly botService: BotService
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
            case 'error': {
                bot.answerCallbackQuery(callbackQuery.id, {
                    text: 'Напишите пожалуйста ошибку',
                })
                await this.botService.update(callbackQuery.message.chat.id, {
                    waitingFor: 'error',
                    msg_id: callbackQuery.message.message_id,
                })
                await bot
                    .editMessageText(
                        '⚠️ <b>Опишите ошибку, с которой вы столкнулись:</b>\n\n' +
                            'Пожалуйста, подробно опишите, что именно пошло не так. 🤔\n' +
                            'Укажите, на каком этапе произошла ошибка, что вы делали перед этим и что именно не сработало.\n' +
                            'Чем подробнее будет ваш ответ — тем быстрее мы сможем всё исправить! 🔧💬' +
                            ' '.repeat(Math.random() * 100),
                        {
                            chat_id: callbackQuery.message.chat.id,
                            message_id: callbackQuery.message.message_id,
                            parse_mode: 'HTML',
                        }
                    )
                    .catch((error) =>
                        this.logger.error(
                            'Error editing message text (error): ' + error
                        )
                    )
                return
            }
        }
    }
}
