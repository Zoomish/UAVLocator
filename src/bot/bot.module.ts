import { forwardRef, Module } from '@nestjs/common'
import { TypeOrmModule } from '@nestjs/typeorm'
import { UserModule } from 'src/user/user.module'
import { BotService } from './bot.service'
import { Bot } from './entities/bot.entity'
import {
    CallbackService,
    GetAdminCallbackService,
    GetAdminService,
    HandleCancelService,
    HandleGetService,
    HandleService,
    HandleSetService,
    InputTextService,
    NoCommandsService,
    NoSessionService,
    SendInfoService,
    SetAdminCallbackService,
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
        SendInfoService,
        GetAdminService,
        CallbackService,
        HandleSetService,
        HandleGetService,
        SetAdminCallbackService,
        GetAdminCallbackService,
        HandleCancelService,
    ],
    exports: [NoSessionService, SendInfoService],
})
export class BotModule {}
