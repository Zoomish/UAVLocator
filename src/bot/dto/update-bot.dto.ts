import { IsOptional } from 'class-validator'

export class UpdateBotDto {
    @IsOptional()
    msg_id?: number

    @IsOptional()
    waitingFor?: string

    @IsOptional()
    msg_id_delete?: number
}
