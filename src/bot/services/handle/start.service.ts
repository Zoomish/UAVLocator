import { forwardRef, Inject, Injectable, Logger } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'
import { UserService } from 'src/user/user.service'

@Injectable()
export class StartService {
    constructor(
        @Inject(forwardRef(() => UserService))
        private readonly userService: UserService
    ) {}
    private readonly logger = new Logger(StartService.name)

    async startMessage() {
        const bot: TelegramBot = global.bot
        const msg: TelegramBot.Message = global.msg
        let user = await this.userService.findOne(msg.chat.id)

        if (user) {
            user = await this.userService.update(msg.chat.id, {
                username: msg.chat.username,
                name: msg.chat.first_name,
            })
        }else{
            user = await this.userService.create({
                tgId: msg.chat.id,
                username: msg.chat.username,
                name: msg.chat.first_name,
            })
        }

        await bot
            .sendMessage(
                msg.chat.id,
                `Добро пожаловать в <b>Локатор России</b>! 🛡️\n\n` +
                    `Это ваш персональный помощник для отслеживания угрозы БПЛА в режиме реального времени.\n\n` +
                    `<b>Как это работает:</b>\n` +
                    `• Бот мониторит указанные вами <b>локации</b>.\n` +
                    `• При появлении опасности вы мгновенно получаете <b>уведомление</b>.\n\n` +
                    `<b>Для начала работы:</b>\n` +
                    `1. Перейдите в <b>Настройки</b> (кнопка ниже).\n` +
                    `2. Укажите регионы, которые хотите отслеживать.\n` +
                    `<i>Будьте в курсе. Оставайтесь в безопасности.</i>\n\n` +
                    `<b>Доступные команды:</b>\n` +
                    `• <code>/start</code> — показать это меню\n` +
                    `• <code>/settings</code> — открыть настройки\n\n` +
                    `<b>Основная кнопка:</b>\n` +
                    `⚙️ <b>Настройки</b> — выбрать локации и управлять уведомлениями.`,
                {
                    parse_mode: 'HTML',
                    reply_markup: {
                        inline_keyboard: [
                            [
                                {
                                    text: '⚙️ Настройки',
                                    callback_data: 'get_settings',
                                },
                            ],
                        ],
                    },
                }
            )
            .catch((error) =>
                this.logger.error('Error deleting start message: ' + error)
            )

        await bot
            .deleteMessage(msg.chat.id, msg.message_id)
            .catch((error) =>
                this.logger.error('Error deleting start message: ' + error)
            )
    }
}
