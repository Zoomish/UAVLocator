import { Injectable, Logger } from '@nestjs/common'
import { Interval } from '@nestjs/schedule'
import { UserService } from '../user/user.service'
import { ConfigService } from '@nestjs/config'

@Injectable()
export class CheckMessagesService {
    constructor(private readonly userService: UserService) {}
    private readonly logger = new Logger(CheckMessagesService.name)

    @Interval(1000 * 60)
    async handleTimeout() {
        await this.userService.checkLocations()
        this.logger.log('Messages checked')
    }
}
