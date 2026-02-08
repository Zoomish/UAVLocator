import { forwardRef, Module } from '@nestjs/common'
import { TypeOrmModule } from '@nestjs/typeorm'
import { BotModule } from 'src/bot/bot.module'
import { Bot } from 'src/bot/entities/bot.entity'
import { ChannelService } from 'src/user/telegram/channel.service'
import { User } from './entities/user.entity'
import { TelegramClientService } from './telegram/telegramClient.service'
import { TelegramLoggerService } from './telegram/telegramLogger.service'
import { UserService } from './user.service'

@Module({
    imports: [
        TypeOrmModule.forFeature([User, Bot]),
        forwardRef(() => BotModule),
    ],
    providers: [
        UserService,
        ChannelService,
        TelegramClientService,
        TelegramLoggerService,
    ],
    exports: [UserService],
})
export class UserModule {}
