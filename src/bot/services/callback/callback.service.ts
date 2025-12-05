import { Injectable } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'

@Injectable()
export class CallbackService {
    constructor() {}
    async handleCallback(callbackQuery: TelegramBot.CallbackQuery) {
        const text = callbackQuery.data
        global.msg = callbackQuery.message

        switch (text) {
            case 'set':
            case 'get':
        }
    }
}
