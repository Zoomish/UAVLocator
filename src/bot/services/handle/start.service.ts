import { Injectable, Logger } from '@nestjs/common'
import { ConfigService } from '@nestjs/config'
import TelegramBot from 'node-telegram-bot-api'
import { BotService } from 'src/bot/bot.service'
import { UserService } from 'src/user/user.service'

@Injectable()
export class StartService {
    private readonly logger = new Logger(StartService.name)

    constructor(
        private readonly userService: UserService,
        private readonly botService: BotService,
        private readonly configService: ConfigService
    ) {}

    async startMessage() {
        const bot: TelegramBot = global.bot
        const msg: TelegramBot.Message = global.msg
        let user = await this.userService.findOne(msg.chat.id)

        if (user) {
            user = await this.userService.update(msg.chat.id, {
                username: msg.chat.username,
                name: msg.chat.first_name,
            })
        } else {
            const today = new Date()
            const nextThreeDays = new Date(today.setDate(today.getDate() + 3))
            user = await this.userService.create({
                tgId: msg.chat.id,
                username: msg.chat?.username || null,
                name: msg.chat?.first_name || null,
                expires_in: nextThreeDays,
            })
        }

        await bot
            .deleteMessage(msg.chat.id, msg.message_id)
            .catch((error) =>
                this.logger.error('Error deleting start message: ' + error)
            )

        try {
            if (user.hh?.resume_id) {
                return await this.readyWorkMessage(msg.chat.id, bot)
            }
            if (user.hh?.access_token) {
                return await this.chooseResumeMessage(msg.chat.id, bot)
            }
            return await this.defultMessage(msg.chat.id, bot)
        } catch (error) {
            this.logger.error('Error sending start response: ' + error)
        }
    }

    private async defultMessage(chatId: number, bot: TelegramBot) {
        return await bot
            .sendMessage(
                chatId,
                `<b>🎉 Добро пожаловать! Рад видеть тебя здесь! 😊</b>\n\n` +
                    `<i>У меня для тебя есть особый подарок 🎁 — 3 дня бесплатной подписки!</i>\n\n` +
                    `<b>Подписка уже активирована автоматически, и ты можешь начать пользоваться всеми возможностями прямо сейчас!</b>\n\n` +
                    `<i>С этим ботом ты сможешь:</i>\n\n` +
                    `<b>⚙️ Автоматически откликаться на вакансии</b> — бот сам найдёт подходящие предложения и отправит отклики за тебя, экономя часы времени.\n\n` +
                    `<b>✍️ Генерировать профессиональные сопроводительные письма</b> — можно выбрать универсальное письмо или создавать уникальное для каждой вакансии с помощью ИИ.\n\n` +
                    `<b>🔍 Быстро найти работу мечты</b> — с помощью гибких фильтров ты будешь видеть только самые подходящие предложения.\n\n` +
                    `<i>Попробуй прямо сейчас и почувствуй, как легко может быть поиск работы! 🚀</i>\n\n` +
                    `<b>🔐 Чтобы начать, нужно авторизоваться через hh.ru:</b>\n` +
                    `Нажав кнопку ниже, ты перейдёшь на официальный сайт hh.ru и дашь доступ боту к своему аккаунту. Это нужно, чтобы бот мог от твоего имени находить вакансии и отправлять отклики. Все данные защищены и используются только для работы сервиса.`,
                {
                    parse_mode: 'HTML',
                    reply_markup: {
                        inline_keyboard: [
                            [
                                {
                                    text: 'Авторизоваться',
                                    url:
                                        `${this.configService.get('SERVER_URL')}/api/redirect/hhlogin?tgId=` +
                                        chatId,
                                },
                            ],
                        ],
                    },
                }
            )
            .catch((error) =>
                this.logger.error(
                    'Error sending default start message: ' + error
                )
            )
    }

    private async chooseResumeMessage(chatId: number, bot: TelegramBot) {
        return await bot
            .sendMessage(
                chatId,
                '<b>И снова привет! Рад видеть тебя здесь! 😊</b>\n\n' +
                    '<i>Для начала работы осталось только выбрать резюме для откликов!</i> 📄\n\n' +
                    '<b>Что делать дальше?</b>\n' +
                    'Нажми на кнопку <i>«Выбрать резюме»</i> ниже, чтобы открыть список твоих резюме и выбрать то, с которым хочешь работать.\n\n' +
                    '📄 <b>Выбрать резюме</b> — показывает список всех твоих резюме, из которого ты сможешь легко выбрать нужное.\n' +
                    'Это важный шаг, потому что бот будет использовать выбранное резюме для автоматических откликов на вакансии и сэкономит тебе много времени! ⏳',
                {
                    parse_mode: 'HTML',
                    reply_markup: {
                        inline_keyboard: [
                            [
                                {
                                    text: '📄 Выбрать резюме',
                                    callback_data: 'choose_resume',
                                },
                            ],
                        ],
                    },
                }
            )
            .catch((error) =>
                this.logger.error(
                    'Error sending choose resume message: ' + error
                )
            )
    }

    private async readyWorkMessage(chatId: number, bot: TelegramBot) {
        return await bot
            .sendMessage(
                chatId,
                '👋 <b>С возвращением! 😊</b>\n' +
                    'Рады видеть вас снова! Вы уже успешно зарегистрированы на hh.ru и выбрали резюме для откликов. Теперь можно продолжать работать и находить лучшие вакансии вместе с нашим ботом! 🚀\n\n' +
                    '<b>Вот что вы можете сделать дальше:</b>\n\n' +
                    '⚙️ <b>Настройки</b> — перейдите сюда, чтобы обновить фильтры вакансий, сменить резюме для откликов или отредактировать сопроводительное письмо. Всё, чтобы сделать отклики максимально персонализированными и эффективными! 🛠️\n\n' +
                    '🤖 <b>Автоотклики</b> — здесь вы можете запустить автоматическую рассылку откликов на подходящие вакансии или продлить подписку для продолжения работы бота без перерывов. Больше откликов — больше шансов найти работу мечты! 💼✨',
                {
                    parse_mode: 'HTML',
                    reply_markup: {
                        inline_keyboard: [
                            [
                                {
                                    text: '⚙️ Настройки',
                                    callback_data: 'get_settings',
                                },
                                {
                                    text: '🤖 Автоотклики',
                                    callback_data: 'get_apply',
                                },
                            ],
                        ],
                    },
                }
            )
            .catch((error) =>
                this.logger.error('Error sending ready work message: ' + error)
            )
    }
}
