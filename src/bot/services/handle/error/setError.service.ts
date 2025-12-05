import { Injectable, Logger } from '@nestjs/common'
import TelegramBot, { Message } from 'node-telegram-bot-api'
import { BotService } from 'src/bot/bot.service'

@Injectable()
export class HandleErrorService {
    constructor(private readonly botService: BotService) {}

    private readonly logger = new Logger(HandleErrorService.name)

    async handleError() {
        const bot: TelegramBot = global.bot
        const msg: TelegramBot.Message = global.msg
        await bot
            .deleteMessage(msg.chat.id, msg.message_id)
            .catch((error) =>
                this.logger.error('Error deleting start message: ' + error)
            )
        const msgSent = (await bot
            .sendMessage(
                msg.chat.id,
                '⚠️ <b>Опишите ошибку, с которой вы столкнулись:</b>\n\n' +
                    'Пожалуйста, подробно опишите, что именно пошло не так. 🤔\n' +
                    'Укажите, на каком этапе произошла ошибка, что вы делали перед этим и что именно не сработало.\n' +
                    'Чем подробнее будет ваш ответ — тем быстрее мы сможем всё исправить! 🔧💬' +
                    ' '.repeat(Math.random() * 100),
                {
                    parse_mode: 'HTML',
                    reply_markup: {
                        inline_keyboard: [
                            [
                                {
                                    text: '❌ Отменить',
                                    callback_data: 'cancel_error',
                                },
                            ],
                        ],
                    },
                }
            )
            .catch((error) =>
                this.logger.error(
                    'Error editing message text (error): ' + error
                )
            )) as Message
        await this.botService.update(msg.chat.id, {
            waitingFor: 'error',
            msg_id: msgSent.message_id,
        })
    }
}
