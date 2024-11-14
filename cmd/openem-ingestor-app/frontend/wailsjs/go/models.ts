export namespace main {
	
	export class ExtractionMethod {
	
	
	    static createFrom(source: any = {}) {
	        return new ExtractionMethod(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	
	    }
	}

}

