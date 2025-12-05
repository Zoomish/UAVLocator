import { forwardRef, Inject, Injectable, Logger } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'
import { UserService } from 'src/user/user.service'

@Injectable()
export class SettingsService {
    constructor(
        @Inject(forwardRef(() => UserService))
        private readonly userService: UserService
    ) {}
    private readonly logger = new Logger(SettingsService.name)

    async settings(msgId?: number) {
        const bot: TelegramBot = global.bot
        const msg: TelegramBot.Message = global.msg
        const user = await this.userService.findOne(msg.chat.id)
        const text =
            `⚙️ <b>Настройки</b>\n\n` +
            `Здесь вы можете настроить параметры получения уведомлений о БПЛА.\n\n` +
            `<b>Текущие настройки:</b>\n` +
            `📍 <b>Отслеживаемые локации</b>: ${user.locations.join(', ')}\n` +
            `${user.notifications ? '🔔' : '🔕'} <b>Уведомления</b>: ${user.notifications ? 'включены' : 'отключены'}\n\n` +
            `<b>Что вы можете изменить:</b>\n` +
            `• Выбрать конкретные регионы для отслеживания угроз\n` +
            `• Включить или отключить получение уведомлений\n` +
            `• Изменить список локаций в любое время\n\n` +
            `<b>Кнопки настроек:</b>\n` +
            `📍 <b>Локации</b> — изменить список отслеживаемых регионов\n` +
            `${user.notifications ? '🔕' : '🔔'} <b>Уведомления</b> — ${user.notifications ? 'отключить' : 'включить'} оповещения о новых угрозах`
        const reply_markup = {
            inline_keyboard: [
                [
                    {
                        text: '📍 Локации',
                        callback_data: 'set_locations',
                    },
                ],
                [
                    {
                        text: `${user.notifications ? '🔕' : '🔔'} Уведомления`,
                        callback_data:
                            'set_notifications-' +
                            (user.notifications ? 'false' : 'true'),
                    },
                ],
                user.admin
                    ? [{ text: '🛠️ Админка', callback_data: 'get_admin' }]
                    : [],
            ],
        }
        if (msgId) {
            return await bot
                .editMessageText(text, {
                    chat_id: msg.chat.id,
                    message_id: msgId,
                    parse_mode: 'HTML',
                    reply_markup: reply_markup,
                })
                .catch((error) =>
                    this.logger.error('Error editing message: ' + error)
                )
        }
        await bot
            .deleteMessage(msg.chat.id, msg.message_id)
            .catch((error) =>
                this.logger.error('Error deleting message: ' + error)
            )
        return await bot
            .sendMessage(msg.chat.id, text, {
                parse_mode: 'HTML',
                reply_markup: reply_markup,
            })
            .catch((error) =>
                this.logger.error('Error sending message: ' + error)
            )
    }
}
