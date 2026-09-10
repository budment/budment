import { Scenario as ProtoScenario, ScenarioConfig } from '../pb/ast_schema';
import { Pipeline } from './pipeline';

/**
 * Root builder for scenario definition and configuration.
 */
export class ScenarioBuilder {
    private scenarioName: string;
    private scenarioConfig?: ScenarioConfig;
    private setupPipeline = new Pipeline();
    private executionPipeline = new Pipeline();

    constructor(name: string) {
        this.scenarioName = name;
    }

    config(cfg: ScenarioConfig): this {
        this.scenarioConfig = cfg;
        return this;
    }

    setup(...builders: any[]): this {
        this.setupPipeline.add(...builders);
        return this;
    }

    execution(...builders: any[]): this {
        this.executionPipeline.add(...builders);
        return this;
    }

    build(configOverride?: ScenarioConfig): ProtoScenario {
        return {
            name: this.scenarioName,
            setup: this.setupPipeline.build(),
            execution: this.executionPipeline.build(),
            config: configOverride || this.scenarioConfig
        };
    }
}

export function scenario(name: string): ScenarioBuilder {
    return new ScenarioBuilder(name);
}
