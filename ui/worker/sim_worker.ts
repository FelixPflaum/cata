import { WorkerInterface } from './worker_interface';

type SimRequestAsync = (data: Uint8Array, progress: (result: Uint8Array) => void, id: string) => Uint8Array;
type SimRequestSync = (data: Uint8Array) => Uint8Array;

// Functions provided or used by the wasm lib.
declare global {
	function wasmready(): void;
	const bulkSimAsync: SimRequestAsync;
	const bulkSimCombos: SimRequestSync;
	const computeStats: SimRequestSync;
	const computeStatsJson: SimRequestSync;
	const raidSim: SimRequestSync;
	const raidSimJson: SimRequestSync;
	const raidSimAsync: SimRequestAsync;
	const statWeights: SimRequestSync;
	const statWeightsAsync: SimRequestAsync;
	const statWeightRequests: SimRequestSync;
	const statWeightCompute: SimRequestSync;
	const raidSimResultCombination: SimRequestSync;
	const raidSimRequestSplit: SimRequestSync;
	const abortById: SimRequestSync;
}

// Wasm binary calls this function when its done loading.
// eslint-disable-next-line @typescript-eslint/no-unused-vars
globalThis.wasmready = function () {
	new WorkerInterface({
		bulkSimAsync: bulkSimAsync,
		bulkSimCombos: bulkSimCombos,
		computeStats: computeStats,
		computeStatsJson: computeStatsJson,
		raidSim: raidSim,
		raidSimJson: raidSimJson,
		raidSimAsync: raidSimAsync,
		statWeights: statWeights,
		statWeightsAsync: statWeightsAsync,
		statWeightRequests: statWeightRequests,
		statWeightCompute: statWeightCompute,
		raidSimRequestSplit: raidSimRequestSplit,
		raidSimResultCombination: raidSimResultCombination,
		abortById: abortById,
	}).ready(true);
};

const go = new Go();
let inst: WebAssembly.Instance | null = null;

function getURL() {
	const rnd = Math.random();
	if (rnd > 0.8) return "kekw.wasm";
	return "lib.wasm";
}

WebAssembly.instantiateStreaming(fetch(getURL()), go.importObject)
	.then(async result => {
		inst = result.instance;
		await go.run(inst);
	})
	.catch(error => {
		console.error(error);
		if (error instanceof Error) {
			WorkerInterface.error(error);
		} else {
			WorkerInterface.error(new Error('unknown error'));
		}
	});

export {};
