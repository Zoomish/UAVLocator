import { forwardRef, Inject, Injectable, Logger } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'
import { BotService } from 'src/bot/bot.service'
import { UserService } from 'src/user/user.service'
import { SettingsService } from './settings.service'

@Injectable()
export class InputTextService {
    constructor(
        private readonly botService: BotService,
        @Inject(forwardRef(() => UserService))
        private readonly userService: UserService,

        @Inject(forwardRef(() => SettingsService))
        private readonly settingsService: SettingsService
    ) {}
    private readonly logger = new Logger(InputTextService.name)

    async handleInputText(text: string) {
        const bot: TelegramBot = global.bot
        const msg: TelegramBot.Message = global.msg

        try {
            const botService = await this.botService.findOne(msg.chat.id)
            const texts = botService.waitingFor?.split('=') || []

            switch (texts[0]) {
                case 'error': {
                    const msgId = botService.msg_id
                    await this.botService.update(msg.chat.id, {
                        waitingFor: null,
                        msg_id: null,
                    })
                    await bot
                        .deleteMessage(msg.chat.id, msg.message_id)
                        .catch((error) =>
                            this.logger.error(
                                'Failed to delete message (error): ' + error
                            )
                        )

                    const admin = await this.userService.findAdmin()
                    if (admin) {
                        await bot
                            .sendMessage(
                                admin.tgId,
                                '⚠️ Ошибка!\n' +
                                    `@${msg.from.username} (${msg.chat.id}):\n` +
                                    text
                            )
                            .catch((error) =>
                                this.logger.error(
                                    'Failed to send error message to admin: ' +
                                        error
                                )
                            )
                    } else {
                        this.logger.warn('No admin found to send error message')
                    }

                    await bot
                        .editMessageText(
                            '🙏 <b>Спасибо за обратную связь!</b> 😊\n' +
                                'Мы очень ценим ваше участие и обязательно исправим найденную ошибку как можно скорее! 🛠️✨\n' +
                                'Если у вас появятся ещё замечания или предложения — не стесняйтесь писать нам!\n\n' +
                                '<b>Чтобы продолжить, воспользуйтесь кнопкой ниже:</b>\n' +
                                '⚙️ <b>Настройки</b> — перейдите в раздел настроек, где сможете изменить фильтры, выбрать резюме для откликов или отредактировать сопроводительное письмо для максимального комфорта и эффективности! 🎯' +
                                ' '.repeat(Math.random() * 100),
                            {
                                chat_id: msg.chat.id,
                                message_id: msgId,
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
                            this.logger.error(
                                'Failed to edit message text (error): ' + error
                            )
                        )
                    break
                }
                case 'locations': {
                    await bot
                        .deleteMessage(msg.chat.id, msg.message_id)
                        .catch((error) =>
                            this.logger.error(
                                'Failed to delete message (error): ' + error
                            )
                        )
                    await this.userService.update(msg.chat.id, {
                        locations: text.split(',').map(this.normalizeText),
                    })
                    await this.settingsService.settings(botService.msg_id)
                    break
                }
                default: {
                    this.logger.warn(`Unknown waitingFor prefix: ${texts[0]}`)
                }
            }

            if (botService.msg_id_delete) {
                await bot
                    .deleteMessage(msg.chat.id, botService.msg_id_delete)
                    .catch((error) =>
                        this.logger.error(
                            'Failed to delete message (msg_id_delete): ' + error
                        )
                    )
            }

            await this.botService.update(msg.chat.id, {
                waitingFor: null,
                msg_id: null,
                msg_id_delete: null,
            })
        } catch (error) {
            this.logger.error('Error in handleInputText: ' + error)
        }
    }

    private normalizeText(text: string): string {
        return text
            .toLowerCase()
            .replace(/[^а-я0-9\s]/g, ' ')
            .replace(/\s+/g, ' ')
            .trim()
    }
}
