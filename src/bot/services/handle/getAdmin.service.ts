import { forwardRef, Inject, Injectable, Logger } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'
import { UserService } from 'src/user/user.service'

@Injectable()
export class GetAdminService {
    constructor(
        @Inject(forwardRef(() => UserService))
        private readonly userService: UserService
    ) {}
    private readonly logger = new Logger(GetAdminService.name)

    async getAdmin(msgId?: number) {
        const bot: TelegramBot = global.bot
        const msg: TelegramBot.Message = global.msg
        const users = await this.userService.findAll()
        const text =
            'Панель администратора:\n\n' +
            `👥 <b>Всего пользователей</b>: ${users.length}\n` +
            ' '.repeat(Math.random() * 100)
        const reply_markup = {
            inline_keyboard: [
                [
                    {
                        text: 'Получить всех пользователей',
                        callback_data: 'get_admin-users',
                    },
                ],
                [
                    {
                        text: 'Отправить объявление',
                        callback_data: 'get_admin-announce',
                    },
                ],
                [
                    {
                        text: '⚙️ Настройки',
                        callback_data: 'get_settings',
                    },
                ],
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
        } else {
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
}
