import { forwardRef, Module } from '@nestjs/common'
import { TypeOrmModule } from '@nestjs/typeorm'
import { BotModule } from 'src/bot/bot.module'
import { Bot } from 'src/bot/entities/bot.entity'
import { User } from './entities/user.entity'
import { UserService } from './user.service'

@Module({
    imports: [
        TypeOrmModule.forFeature([User, Bot]),
        forwardRef(() => BotModule),
    ],
    providers: [UserService],
    exports: [UserService],
})
export class UserModule {}
