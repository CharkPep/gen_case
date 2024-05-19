import Redis from "ioredis";

export const rdb = new Redis(process.env["REDIS_URL"] ?? "")
export const GetPush = (queue: string) => {
    return async (msg: any) => {
        await rdb.xadd(queue, "*", ...Object.getOwnPropertyNames(msg).reduce<any[]>((acc, currentValue) => {
            if (!Number.isNaN(msg[currentValue])) {
                acc.push(currentValue, msg[currentValue])
                return acc
            }
            
            acc.push(currentValue, -1)
            return acc
       }, []))
    }
}
export const GetBulkPush = (queue: string) => {
    return async (messages: Record<string, any>[]) => {
        const p = rdb.pipeline()
        for (const msg of messages) {
            p.xadd(queue, "*", ...Object.getOwnPropertyNames(msg).reduce<any[]>((acc, currentValue) => {
                if (!Number.isNaN(msg[currentValue])) {
                    if (msg[currentValue] instanceof Date) {
                        acc.push(currentValue, (msg[currentValue] as Date).toISOString())
                        return acc
                    }
                    acc.push(currentValue, msg[currentValue])
                    return acc
                }

                acc.push(currentValue, null)
                return acc
            }, []))
        }

        await p.exec()
    }
}
