import { Injectable, Logger } from '@nestjs/common'
import TelegramBot from 'node-telegram-bot-api'
import { BotService } from 'src/bot/bot.service'

@Injectable()
export class NoCommandsService {
    constructor(private readonly botService: BotService) {}
    private readonly logger = new Logger(NoCommandsService.name)

    async NoCommands() {
        const bot: TelegramBot = global.bot
        const msg: TelegramBot.Message = global.msg
        await bot
            .deleteMessage(msg.chat.id, msg.message_id)
            .catch((error) =>
                this.logger.error('Error deleting message: ' + error)
            )
        const botService = await this.botService.findOne(msg.chat.id)
        if (botService.msg_id_delete) {
            return
        }
        const msgSent = (await bot
            .sendMessage(
                msg.chat.id,
                '<b>⚠️ Внимание!</b> ❌\n' +
                    'В полях ввода нельзя использовать команды или специальные строки.\n' +
                    'Пожалуйста, попробуйте ввести информацию ещё раз, избегая подобных символов или фраз.\n',
                {
                    parse_mode: 'HTML',
                }
            )
            .catch((error) =>
                this.logger.error('Error sending message: ' + error)
            )) as TelegramBot.Message

        return await this.botService.update(msg.chat.id, {
            msg_id_delete: msgSent?.message_id,
        })
    }
}
