import {
    Column,
    CreateDateColumn,
    Entity,
    PrimaryGeneratedColumn,
} from 'typeorm'

@Entity()
export class DbKeepAlive {
    @PrimaryGeneratedColumn()
    id: number

    @Column({ type: 'varchar', length: 64 })
    payload: string

    @CreateDateColumn()
    createdAt: Date
}
