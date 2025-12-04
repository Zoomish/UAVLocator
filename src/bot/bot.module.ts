import { forwardRef, Module } from '@nestjs/common'
import { TypeOrmModule } from '@nestjs/typeorm'
import { UserModule } from 'src/user/user.module'
import { BotService } from './bot.service'
import { Bot } from './entities/bot.entity'
import {
    HandleService,
    InputTextService,
    NoCommandsService,
    NoSessionService,
    SettingsService,
    StartService,
} from './services'

@Module({
    imports: [TypeOrmModule.forFeature([Bot]), forwardRef(() => UserModule)],
    providers: [
        BotService,
        HandleService,
        StartService,
        SettingsService,
        InputTextService,
        NoCommandsService,
        NoSessionService,
    ],
    exports: [NoSessionService],
})
export class BotModule {}
