import { Scenario as ProtoScenario, ScenarioConfig } from '../pb/ast_schema';
import { Pipeline } from './pipeline';

/**
 * Root builder for scenario definition and configuration.
 */
export class ScenarioBuilder {
    private scenarioName: string;
    private setupPipeline = new Pipeline();
    private executionPipeline = new Pipeline();
    private isRegistered = false;

    constructor(name: string) {
        this.scenarioName = name;
    }

    setup(...builders: any[]): this {
        this.setupPipeline.add(...builders);
        return this;
    }

    execution(...builders: any[]): this {
        this.executionPipeline.add(...builders);
        return this;
    }

    build(config?: ScenarioConfig): ProtoScenario {
        const scn: ProtoScenario = {
            name: this.scenarioName,
            setup: this.setupPipeline.build(),
            execution: this.executionPipeline.build(),
            config: config
        };

        // Dedup Guard: Ensure the scenario is registered exactly once in the global list
        if (!this.isRegistered) {
            const _global = globalThis as any;
            _global.__BUDMENT_SCENARIOS__ = _global.__BUDMENT_SCENARIOS__ || [];

            // Check if scenario with identical name is already registered
            const exists = _global.__BUDMENT_SCENARIOS__.some(
                (existing: ProtoScenario) => existing.name === this.scenarioName
            );
            if (!exists) {
                _global.__BUDMENT_SCENARIOS__.push(scn);
            }
            this.isRegistered = true;
        }

        return scn;
    }
}

/**
 * Fluent builder initializer for declarative scenarios.
 */
export function scenario(name: string): ScenarioBuilder {
    return new ScenarioBuilder(name);
}
