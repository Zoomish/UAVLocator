import { forwardRef, Inject, Injectable, Logger } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'
import { UserService } from 'src/user/user.service'

@Injectable()
export class SettingsService {
    constructor(
        private readonly userService: UserService
    ) {}
    private readonly logger = new Logger(SettingsService.name)

    async settings(msgId?: number) {
        const bot: TelegramBot = global.bot
        const msg: TelegramBot.Message = global.msg
        const user = await this.userService.findOne(msg.chat.id)
        const text =
            '⚙️ <b>Настройки</b>:\n\n' +
            '\n\n' +
            '🤖 <b>Автоотклики</b> — меню, где вы можете запустить автоматическую рассылку откликов на подходящие вакансии или продлить подписку, чтобы не прерывать работу бота. Больше откликов — больше шансов на успех! 💼✨' +
            ' '.repeat(Math.random() * 100)
        const reply_markup = {
            inline_keyboard: [
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
