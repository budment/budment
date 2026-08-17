import { Scenario as ProtoScenario, ScenarioConfig } from "../../pb/ast_schema";
import { Pipeline } from "./pipeline";
import { BuilderNode } from "./types";

/**
 * Root builder for scenario definition and configuration.
 */
export class ScenarioBuilder {
    private scenarioName: string;
    private setupPipeline = new Pipeline();
    private executionPipeline = new Pipeline();
    private execConfig?: ScenarioConfig;

    constructor(name: string) {
        this.scenarioName = name;
    }

    setup(...builders: BuilderNode[]): this {
        this.setupPipeline.add(...builders);
        return this;
    }

    execution(...builders: BuilderNode[]): this {
        this.executionPipeline.add(...builders);
        return this;
    }

    use(config: ScenarioConfig): this {
        this.execConfig = config;
        return this;
    }

    build(): ProtoScenario {
        return {
            name: this.scenarioName,
            setup: this.setupPipeline.build(),
            execution: this.executionPipeline.build(),
            config: this.execConfig,
        };
    }
}

export function scenario(name: string) {
    return new ScenarioBuilder(name);
}
