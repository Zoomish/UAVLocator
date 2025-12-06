import { forwardRef, Inject, Injectable, Logger } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'
import { BotService } from 'src/bot/bot.service'
import { HandleCancelService } from './canсel'
import { HandleGetService } from './get'
import { HandleSetService } from './set'

@Injectable()
export class CallbackService {
    constructor(
        @Inject(forwardRef(() => BotService))
        private readonly botService: BotService,
        private readonly handleSetService: HandleSetService,
        private readonly handleGetService: HandleGetService,
        private readonly handleCancelService: HandleCancelService
    ) {}
    private readonly logger = new Logger(CallbackService.name)

    async handleCallback(callbackQuery: TelegramBot.CallbackQuery) {
        const texts = [
            callbackQuery.data.split('_')[0],
            callbackQuery.data.split('_').slice(1).join('_'),
        ]
        global.msg = callbackQuery.message

        const botService = await this.botService.findOne(
            callbackQuery.message.chat.id
        )
        if (botService.waitingFor !== null && texts[0] !== 'cancel') {
            const bot: TelegramBot = global.bot
            return bot
                .answerCallbackQuery(callbackQuery.id, {
                    text: 'Бот пока недоступен для взаимодействия. Подождите пожалуйста',
                })
                .catch((error) =>
                    this.logger.error('Error answering callback: ' + error)
                )
        }

        switch (texts[0]) {
            case 'set':
                return await this.handleSetService.handleSet(
                    texts[1],
                    callbackQuery
                )
            case 'get':
                return await this.handleGetService.handleGet(
                    texts[1],
                    callbackQuery
                )
            case 'cancel':
                return await this.handleCancelService.handleCancel(
                    texts[1],
                    callbackQuery
                )
        }
    }
}
