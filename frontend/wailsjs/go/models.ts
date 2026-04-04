export namespace main {
	
	export class AuthStatus {
	    authenticated: boolean;
	    source?: string;
	
	    static createFrom(source: any = {}) {
	        return new AuthStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.authenticated = source["authenticated"];
	        this.source = source["source"];
	    }
	}
	export class DeviceFlowState {
	    deviceCode: string;
	    userCode: string;
	    verificationURI: string;
	    expiresIn: number;
	    interval: number;
	
	    static createFrom(source: any = {}) {
	        return new DeviceFlowState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.deviceCode = source["deviceCode"];
	        this.userCode = source["userCode"];
	        this.verificationURI = source["verificationURI"];
	        this.expiresIn = source["expiresIn"];
	        this.interval = source["interval"];
	    }
	}

}

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

