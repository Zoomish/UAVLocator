import { Module } from '@nestjs/common'
import { TypeOrmModule } from '@nestjs/typeorm'
import { Bot } from 'src/bot/entities/bot.entity'
import { DiscountModule } from 'src/discount/discount.module'
import { User } from './entities/user.entity'
import { UserService } from './user.service'

@Module({
    imports: [TypeOrmModule.forFeature([User, Bot]), DiscountModule],
    providers: [UserService],
    exports: [UserService],
})
export class UserModule {}
