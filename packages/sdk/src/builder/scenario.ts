import { Scenario as ProtoScenario, ScenarioConfig } from "../pb/ast_schema";
import { Pipeline } from "./pipeline";
import { BuilderNode } from "./types";

/**
 * Root builder for scenario definition and configuration.
 */
export class ScenarioBuilder {
    private scenarioName: string;
    private setupPipeline = new Pipeline();
    private executionPipeline = new Pipeline();

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

    build(config?: ScenarioConfig): ProtoScenario {
        const scn = {
            name: this.scenarioName,
            setup: this.setupPipeline.build(),
            execution: this.executionPipeline.build(),
            config: config,
        };

        const _global = globalThis as any;
        _global.__BLASTER_SCENARIOS__ = _global.__BLASTER_SCENARIOS__ || [];
        _global.__BLASTER_SCENARIOS__.push(scn);

        return scn;
    }
}

export function scenario(name: string) {
    return new ScenarioBuilder(name);
}
