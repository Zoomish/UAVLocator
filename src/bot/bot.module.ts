import { Module } from '@nestjs/common'
import { TypeOrmModule } from '@nestjs/typeorm'
import { UserModule } from 'src/user/user.module'
import { BotService } from './bot.service'
import { Bot } from './entities/bot.entity'
import {
    HandleService,
    InputTextService,
    NoCommandsService,
    SettingsService,
    StartService,
} from './services'

@Module({
    imports: [TypeOrmModule.forFeature([Bot]), UserModule],
    providers: [
        BotService,
        HandleService,
        StartService,
        SettingsService,
        InputTextService,
        NoCommandsService,
    ],
})
export class BotModule {}
