import { Injectable, Logger } from '@nestjs/common'
import { Interval } from '@nestjs/schedule'
import { UserService } from '../user/user.service'

@Injectable()
export class GetActiveService {
    constructor(private readonly userService: UserService) {}
    private readonly logger = new Logger(GetActiveService.name)

    @Interval(1000 * 60 * 10)
    async handleTimeout() {
        const user = await this.userService.findAdmin()
        if (user) {
            await this.userService.update(user.tgId, { admin: false })
            await this.userService.update(user.tgId, { admin: true })
            this.logger.log('Admin updated')
        }
        this.logger.log('Get server Active!')
    }
}
