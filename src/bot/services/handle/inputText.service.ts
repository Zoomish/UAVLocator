import { forwardRef, Inject, Injectable, Logger } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'
import { BotService } from 'src/bot/bot.service'
import { FilterService } from 'src/filter/filter.service'
import { HHService } from 'src/hh/hh.service'
import { LetterService } from 'src/letter/letter.service'
import { UserService } from 'src/user/user.service'
import { GetAdminCallbackService } from '../callback'
import { GetFiltersService } from './filter'
import { NoCommandsService } from './helpActions'
import { GetLetterService } from './letter'

@Injectable()
export class InputTextService {
    private readonly logger = new Logger(InputTextService.name)

    constructor(
        private readonly botService: BotService,
        @Inject(forwardRef(() => HHService))
        private readonly hhService: HHService,
        private readonly getFiltersService: GetFiltersService,
        private readonly noCommandsService: NoCommandsService,
        private readonly filterService: FilterService,
        private readonly getLetterService: GetLetterService,
        @Inject(forwardRef(() => GetAdminCallbackService))
        private readonly getAdminCallbackService: GetAdminCallbackService,
        @Inject(forwardRef(() => LetterService))
        private readonly letterService: LetterService,
        @Inject(forwardRef(() => UserService))
        private readonly userService: UserService
    ) {}

    async handleInputText(text: string) {
        const bot: TelegramBot = global.bot
        const msg: TelegramBot.Message = global.msg

        try {
            const botService = await this.botService.findOne(msg.chat.id)
            let filters = await this.filterService.getOne(msg.chat.id)
            const dict = await this.hhService.getDictionary()
            const texts = botService.waitingFor?.split('=') || []

            switch (texts[0]) {
                case 'salary': {
                    if (isNaN(+text)) {
                        return await this.noCommandsService.NoCommands()
                    }
                    filters = await this.filterService.update(msg.chat.id, {
                        salary: +text,
                    })
                    const msgId = botService.msg_id
                    await bot
                        .deleteMessage(msg.chat.id, msg.message_id)
                        .catch((error) =>
                            this.logger.error(
                                'Failed to delete message (salary): ' + error
                            )
                        )
                    await this.getFiltersService.getFilters(
                        dict,
                        filters,
                        msgId
                    )
                    break
                }
                case 'text': {
                    console.log(this.quoteKeywords(text))

                    filters = await this.filterService.update(msg.chat.id, {
                        text: this.quoteKeywords(text),
                    })
                    const msgId = botService.msg_id
                    await this.botService.update(msg.chat.id, {
                        waitingFor: null,
                        msg_id: null,
                    })
                    await bot
                        .deleteMessage(msg.chat.id, msg.message_id)
                        .catch((error) =>
                            this.logger.error(
                                'Failed to delete message (text): ' + error
                            )
                        )
                    await this.getFiltersService.getFilters(
                        dict,
                        filters,
                        msgId
                    )
                    break
                }
                case 'staticLetter': {
                    await this.letterService.update(msg.chat.id, {
                        constant: text,
                        pattern: null,
                    })
                    const msgId = botService.msg_id
                    await this.botService.update(msg.chat.id, {
                        waitingFor: null,
                        msg_id: null,
                    })
                    await bot
                        .deleteMessage(msg.chat.id, msg.message_id)
                        .catch((error) =>
                            this.logger.error(
                                'Failed to delete message (staticLetter): ' +
                                    error
                            )
                        )
                    await this.getLetterService.getLetter(msgId)
                    break
                }
                case 'dynamicLetter': {
                    await this.letterService.update(msg.chat.id, {
                        pattern: text,
                        constant: null,
                    })
                    const msgId = botService.msg_id
                    await this.botService.update(msg.chat.id, {
                        waitingFor: null,
                        msg_id: null,
                    })
                    await bot
                        .deleteMessage(msg.chat.id, msg.message_id)
                        .catch((error) =>
                            this.logger.error(
                                'Failed to delete message (dynamicLetter): ' +
                                    error
                            )
                        )
                    await this.getLetterService.getLetter(msgId)
                    break
                }
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
                case 'subscription': {
                    await bot
                        .deleteMessage(msg.chat.id, msg.message_id)
                        .catch((error) =>
                            this.logger.error(
                                'Failed to delete message (subscription): ' +
                                    error
                            )
                        )
                    const user = await this.userService.update(+texts[1], {
                        expires_in: new Date(text),
                    })
                    await this.getAdminCallbackService.sendUserInfo(
                        user,
                        botService.msg_id
                    )
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

    private quoteKeywords(input: string): string {
        const logicMap = {
            и: 'AND',
            или: 'OR',
            не: 'NOT',
        }

        // Удалим лишние пробелы
        input = input.replace(/\s+/g, ' ').trim()

        // Разделим по скобкам с пробелами
        const rawTokens = input
            .replace(/([\(\)])/g, ' $1 ') // добавим пробелы вокруг скобок
            .split(' ')
            .filter(Boolean)

        const tokens = []
        let buffer = []

        for (let token of rawTokens) {
            const low = token.toLowerCase()

            if (low in logicMap) {
                if (buffer.length) {
                    tokens.push(`"${buffer.join(' ')}"`)
                    buffer = []
                }
                tokens.push(logicMap[low]) // заменяем оператор
            } else if (token === '(' || token === ')') {
                if (buffer.length) {
                    tokens.push(`"${buffer.join(' ')}"`)
                    buffer = []
                }
                tokens.push(token)
            } else {
                buffer.push(token)
            }
        }

        if (buffer.length) {
            tokens.push(`"${buffer.join(' ')}"`)
        }

        return tokens.join(' ')
    }
}
