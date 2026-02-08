import { Injectable, Logger } from '@nestjs/common'
import { Interval } from '@nestjs/schedule'
import { UserService } from '../user/user.service'

@Injectable()
export class CheckMessagesService {
    constructor(private readonly userService: UserService) {}
    private readonly logger = new Logger(CheckMessagesService.name)

    @Interval(1000 * 60 * 2)
    async handleTimeout() {
        await this.userService.checkUnreadMessages()
    }
}
