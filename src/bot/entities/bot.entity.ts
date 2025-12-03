import { User } from 'src/user/entities/user.entity'
import {
    Column,
    Entity,
    JoinColumn,
    OneToOne,
    PrimaryGeneratedColumn,
} from 'typeorm'

@Entity()
export class Bot {
    @PrimaryGeneratedColumn()
    id: number

    @Column({ nullable: true, type: 'bigint' })
    msg_id: number

    @Column({ nullable: true })
    waitingFor: string

    @Column({ nullable: true, type: 'bigint' })
    msg_id_delete: number

    @OneToOne(() => User, { nullable: false })
    @JoinColumn({ name: 'tgId', referencedColumnName: 'tgId' })
    user: User
}
