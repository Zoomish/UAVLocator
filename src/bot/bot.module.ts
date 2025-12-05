import { forwardRef, Module } from '@nestjs/common'
import { TypeOrmModule } from '@nestjs/typeorm'
import { UserModule } from 'src/user/user.module'
import { BotService } from './bot.service'
import { Bot } from './entities/bot.entity'
import {
    GetAdminService,
    HandleService,
    InputTextService,
    NoCommandsService,
    NoSessionService,
    SendInfoService,
    SettingsService,
    StartService,
} from './services'
import { CallbackService } from './services/callback'

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
        SendInfoService,
        GetAdminService,
        CallbackService,
    ],
    exports: [NoSessionService, SendInfoService],
})
export class BotModule {}
