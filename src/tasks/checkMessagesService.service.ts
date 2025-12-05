import { Injectable, Logger, OnApplicationBootstrap } from '@nestjs/common'
import { UserService } from '../user/user.service'

@Injectable()
export class CheckMessagesService implements OnApplicationBootstrap {
    constructor(private readonly userService: UserService) {}
    private readonly logger = new Logger(CheckMessagesService.name)

    async onApplicationBootstrap() {
        await this.userService.checkUnreadMessages()
        await this.userService.monitorCommentsInRealTime()
        this.logger.log('Messages checked')
    }
}
