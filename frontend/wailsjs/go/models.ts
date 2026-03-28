export namespace usage {
	
	export class UsageReport {
	    provider: string;
	    periodStart?: string;
	    periodEnd?: string;
	    retrievedAt: string;
	    metrics: Record<string, number>;
	    metadata?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new UsageReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.periodStart = source["periodStart"];
	        this.periodEnd = source["periodEnd"];
	        this.retrievedAt = source["retrievedAt"];
	        this.metrics = source["metrics"];
	        this.metadata = source["metadata"];
	    }
	}

}

