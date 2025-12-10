import { Injectable, Logger, OnApplicationBootstrap } from '@nestjs/common'
import { Interval } from '@nestjs/schedule'
import { UserService } from '../user/user.service'

@Injectable()
export class CheckMessagesService implements OnApplicationBootstrap {
    constructor(private readonly userService: UserService) {}
    private readonly logger = new Logger(CheckMessagesService.name)

    async onApplicationBootstrap() {
        await this.userService.checkUnreadMessages()
    }
    @Interval(1000 * 60 * 2)
    async handleTimeout() {
        await this.userService.checkUnreadMessages()
    }
}
