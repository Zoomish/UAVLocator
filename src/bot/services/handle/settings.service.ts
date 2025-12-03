import { forwardRef, Inject, Injectable, Logger } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'
import { HHService } from 'src/hh/hh.service'
import { UserService } from 'src/user/user.service'

@Injectable()
export class SettingsService {
    constructor(
        @Inject(forwardRef(() => HHService))
        private readonly hhService: HHService,
        private readonly userService: UserService
    ) {}
    private readonly logger = new Logger(SettingsService.name)

    async settings(msgId?: number) {
        const bot: TelegramBot = global.bot
        const msg: TelegramBot.Message = global.msg
        const hh = await this.hhService.findOne(msg.chat.id)
        const user = await this.userService.findOne(msg.chat.id)
        const text =
            '⚙️ <b>Настройки</b>:\n\n' +
            '🔒 <b>Авторизация на hh:</b> ' +
            (hh?.refresh_token ? '✅' : '❌') +
            '\n\n' +
            '⚠️ <b>Сообщить об ошибке</b> — если вы заметили какую-либо проблему или баг в работе бота, нажмите эту кнопку, чтобы быстро отправить нам сообщение и помочь улучшить сервис. 🐞\n\n' +
            '🔍 <b>Фильтры</b> — переход в меню, где вы можете настроить параметры поиска вакансий: требуемый опыт, рабочий график, тип занятости и многое другое. Это поможет получать более релевантные отклики! 🎯\n\n' +
            '✉️ <b>Сопроводительное письмо</b> — здесь можно выбрать тип письма для откликов: <i>статичное</i> (одно и то же для всех вакансий) или <i>динамичное</i> (уникальное сопроводительное письмо для каждого отклика). Настройте письмо под себя! 📝\n\n' +
            '📄 <b>Выбрать резюме</b> — откройте список всех ваших резюме и выберите то, которое бот будет использовать для автоматических откликов. Это важный шаг для корректной работы бота! 📋\n\n' +
            '🤖 <b>Автоотклики</b> — меню, где вы можете запустить автоматическую рассылку откликов на подходящие вакансии или продлить подписку, чтобы не прерывать работу бота. Больше откликов — больше шансов на успех! 💼✨' +
            ' '.repeat(Math.random() * 100)
        const reply_markup = {
            inline_keyboard: [
                [
                    {
                        text: '⚠️ Сообщить об ошибке',
                        callback_data: 'set_error',
                    },
                ],
                [
                    {
                        text: '🔍 Фильтры',
                        callback_data: 'get_filters',
                    },
                    {
                        text: '✉️ Сопр. письмо',
                        callback_data: 'get_letter',
                    },
                ],
                [
                    {
                        text: '📄 Выбрать резюме',
                        callback_data: 'choose_resume',
                    },
                ],
                [
                    {
                        text: '🤖 Автоотклики',
                        callback_data: 'get_apply',
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
