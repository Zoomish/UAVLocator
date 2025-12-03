import { Module } from '@nestjs/common'
import { TypeOrmModule } from '@nestjs/typeorm'
import { Bot } from 'src/bot/entities/bot.entity'
import { User } from './entities/user.entity'
import { UserService } from './user.service'
import { BotModule } from 'src/bot/bot.module'

@Module({
    imports: [TypeOrmModule.forFeature([User, Bot]), BotModule],
    providers: [UserService],
    exports: [UserService],
})
export class UserModule {}
