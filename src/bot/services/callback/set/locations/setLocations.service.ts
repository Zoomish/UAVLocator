import { forwardRef, Inject, Injectable, Logger } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'
import { BotService } from 'src/bot/bot.service'

@Injectable()
export class SetLocationsCallbackService {
    constructor(
        @Inject(forwardRef(() => BotService))
        private readonly botService: BotService
    ) {}
    private readonly logger = new Logger(SetLocationsCallbackService.name)

    async handleSetLocations(callbackQuery: TelegramBot.CallbackQuery) {
        const bot: TelegramBot = global.bot

        bot.answerCallbackQuery(callbackQuery.id, {
            text: 'Перечислите локации',
        }).catch((error) =>
            this.logger.error('Error answering callback: ' + error)
        )
        await this.botService.update(callbackQuery.message.chat.id, {
            waitingFor: 'locations',
            msg_id: callbackQuery.message.message_id,
        })
        await bot
            .editMessageText(
                `📍 <b>Управление локациями</b>\n\n` +
                    `Введите названия регионов, городов или областей для отслеживания через запятую.\n\n` +
                    `<b>Формат ввода:</b>\n` +
                    `<code>Москва, Санкт-Петербург, Краснодарский край, Белгородская область</code>\n\n` +
                    `<b>Принцип работы:</b>\n` +
                    `Уведомление будет приходить, если в оповещении будет упомянута <i>хотя бы одна</i> из указанных вами локаций.\n` +
                    `То есть, поиск работает по принципу <b>"ИЛИ"</b> (OR).\n\n` +
                    `<b>Настоятельно</b> рекомендуется использовать <b>сокращенное название</b> локации, чтобы поиск был более точным.\n` +
                    `<b>Пример:</b> Вместо "Краснодарскии краи" используйте "Краснодар".\n` +
                    `<b>Пример использования:</b>\n` +
                    `Если вы укажете: "Москва, Калуга"\n` +
                    `• Вы получите уведомление, если будет угроза в <b>Москве</b>\n` +
                    `• Вы получите уведомление, если будет угроза в <b>Калуге</b>\n` +
                    `• Вы получите уведомление, если будет угроза в <b>Москве и Калуге</b>\n\n` +
                    `Введите новые локации, чтобы заменить текущий список, или нажмите <b>Отмена</b> для возврата.`,
                {
                    chat_id: callbackQuery.message.chat.id,
                    message_id: callbackQuery.message.message_id,
                    parse_mode: 'HTML',
                    reply_markup: {
                        inline_keyboard: [
                            [
                                {
                                    text: '❌ Отмена',
                                    callback_data: 'cancel_locations',
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
            )
    }
}
