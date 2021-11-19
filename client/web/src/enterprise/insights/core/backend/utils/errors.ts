export class InsightStillProcessingError extends Error {
    constructor(message: string = 'Your insight is being processed') {
        super(message)
        this.name = 'InProcessError'
    }
}
