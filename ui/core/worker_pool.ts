import { SimRequest, WorkerReceiveMessage, WorkerSendMessage } from '../worker/types';
import { REPO_NAME } from './constants/other.js';
import {
	AbortRequest,
	AbortResponse,
	BulkSimCombosRequest,
	BulkSimCombosResult,
	BulkSimRequest,
	BulkSimResult,
	ComputeStatsRequest,
	ComputeStatsResult,
	ProgressMetrics,
	RaidSimRequest,
	RaidSimRequestSplitRequest,
	RaidSimRequestSplitResult,
	RaidSimResult,
	RaidSimResultCombinationRequest,
	StatWeightRequestsData,
	StatWeightsCalcRequest,
	StatWeightsRequest,
	StatWeightsResult,
} from './proto/api.js';
import { SimSignals } from './sim_signal_manager';
import { isDevMode, noop } from './utils';

const SIM_WORKER_URL = `/${REPO_NAME}/sim_worker.js`;
const MAX_WORKER_RESTARTS = 1;
export type WorkerProgressCallback = (progressMetrics: ProgressMetrics) => void;

type WorkerStateChangeCallback = (worker: SimWorker, change: 'ready' | 'disable' | 'error', error?: Error) => void;

/**
 * Create random id for requests.
 * @param type The request type to prepend.
 * @returns Random id in the format type-randomhex
 */
const generateRequestId = (type: SimRequest) => {
	const chars = Array.from(Array(4)).map(() => Math.floor(Math.random() * 0x10000).toString(16));
	return type + '-' + chars.join('');
};

export class WorkerPool {
	private readonly workers: Array<SimWorker>;
	private readonly workersDisabled: Array<SimWorker>;
	private nextWorkerId = 1;
	private errorAlertShown = false;
	private readonly isWasmPromise: Promise<boolean>;
	private isWasmPromiseResolver?: (isWasm: boolean) => void;

	constructor(numWorkers: number) {
		this.workers = [];
		this.workersDisabled = [];
		this.isWasmPromise = new Promise<boolean>(resolve => {
			this.isWasmPromiseResolver = resolve;
		});
		this.setNumWorkers(numWorkers);
	}

	private onWorkerStateChange: WorkerStateChangeCallback = (worker, change, error) => {
		if (change == 'error') {
			const errorString = error instanceof Error ? error.message : 'undefined error';
			console.error(`Worker ${worker.workerId} failed due to an error: ${errorString}`);

			// Attempt reload of worker instance
			if (worker.restartCount < MAX_WORKER_RESTARTS) {
				console.log(`Attempting to reinitialize worker ${worker.workerId}`);
				worker.disable(true);
				worker.restartCount++;
				worker.enable();
				return;
			}

			// This worker failed and can't be saved, oh no!
			// Remove entirely and decrement worker count.
			worker.disable(true);
			for (let i = 0; i < this.workers.length; i++) {
				if (this.workers[i].workerId == worker.workerId) {
					this.workers.splice(i);
					break;
				}
			}

			console.error(`Worker ${worker.workerId} failed to reinitialize after an error, letting it go :(`);

			// TODO: "Temporary" error handling.
			// Users of the workerpool functions should also react to promise rejections
			// and async sim operations should get an error payload in their onProgress() handler.
			//
			// This here is just so users get SOME feedback if workers fail entirely.
			//
			// Could also expose an event and tell worker state/count changes and show worker state in the UI nicely.
			const remainingWorkers = this.workers.length;
			if (remainingWorkers == 0) {
				alert('All workers failed to load, sim will not work!\n\nThe browser console may have more details.\n\nError: ' + errorString);
				return;
			} else if (!this.errorAlertShown) {
				alert(
					'Failed to setup one or more workers! The should still work after this, but it will be slower.\n\nThe browser console may have more details.\n\nError: ' +
						errorString,
				);
				this.errorAlertShown = true;
			}
		} else if (change == 'ready') {
			worker.isWasmWorker().then(this.isWasmPromiseResolver);

			if (worker.restartCount > 0) {
				worker.restartCount = 0;
				console.log(`Worker ${worker.workerId} successfully reinitialized after an error!`);
			}
		}
	};

	async setNumWorkers(numWorkers: number) {
		numWorkers = Math.max(numWorkers, navigator.hardwareConcurrency);

		if (this.workers.length == 0) {
			this.workers[0] = new SimWorker(this.nextWorkerId, this.onWorkerStateChange);
			this.nextWorkerId++;
		}

		// Wait for wasm result of first worker.
		// We can ignore the rest of this if not running wasm.
		if (!(await this.isWasmPromise)) return;

		if (numWorkers < this.workers.length) {
			for (let i = this.workers.length - 1; i >= numWorkers; i--) {
				const removedWorker = this.workers.pop();
				if (removedWorker) {
					removedWorker.disable();
					this.workersDisabled.push(removedWorker);
				}
			}
			return;
		}

		for (let i = 0; i < numWorkers; i++) {
			if (!this.workers[i]) {
				const disabledWorker = this.workersDisabled.pop();
				if (disabledWorker) {
					disabledWorker.enable();
					this.workers[i] = disabledWorker;
				} else {
					this.workers[i] = new SimWorker(this.nextWorkerId, this.onWorkerStateChange);
					this.nextWorkerId++;
				}
			}
		}
	}

	getNumWorkers() {
		return this.workers.length;
	}

	private getLeastBusyWorker(): SimWorker {
		return this.workers.reduce((curMinWorker, nextWorker) =>
			curMinWorker.getSimTaskWorkAmount() < nextWorker.getSimTaskWorkAmount() ? curMinWorker : nextWorker,
		);
	}

	async makeApiCall(requestName: SimRequest, request: Uint8Array): Promise<Uint8Array> {
		return await this.getLeastBusyWorker().doApiCall(requestName, request, generateRequestId(requestName));
	}

	async computeStats(request: ComputeStatsRequest): Promise<ComputeStatsResult> {
		const result = await this.makeApiCall(SimRequest.computeStats, ComputeStatsRequest.toBinary(request));
		return ComputeStatsResult.fromBinary(result);
	}

	private getProgressName(id: string) {
		return `${id}progress`;
	}

	async statWeightsAsync(request: StatWeightsRequest, onProgress: WorkerProgressCallback, signals: SimSignals): Promise<StatWeightsResult> {
		const worker = this.getLeastBusyWorker();
		worker.log('Stat weights request: ' + StatWeightsRequest.toJsonString(request));
		const id = generateRequestId(SimRequest.statWeightsAsync);

		signals.abort.onTrigger(async () => {
			await worker.sendAbortById(id);
		});

		const iterations = request.simOptions ? request.simOptions.iterations * request.statsToWeigh.length : 30000;
		const result = await this.doAsyncRequest(SimRequest.statWeightsAsync, StatWeightsRequest.toBinary(request), id, worker, onProgress, iterations);

		worker.log('Stat weights result: ' + StatWeightsResult.toJsonString(result.finalWeightResult!));
		return result.finalWeightResult!;
	}

	async statWeightRequests(request: StatWeightsRequest): Promise<StatWeightRequestsData> {
		const result = await this.makeApiCall(SimRequest.statWeightRequests, StatWeightsRequest.toBinary(request));
		return StatWeightRequestsData.fromBinary(result);
	}

	async statWeightCompute(request: StatWeightsCalcRequest): Promise<StatWeightsResult> {
		const result = await this.makeApiCall(SimRequest.statWeightCompute, StatWeightsCalcRequest.toBinary(request));
		return StatWeightsResult.fromBinary(result);
	}

	async bulkSimAsync(request: BulkSimRequest, onProgress: WorkerProgressCallback, signals: SimSignals): Promise<BulkSimResult> {
		const worker = this.getLeastBusyWorker();
		worker.log('bulk sim request: ' + BulkSimRequest.toJsonString(request, { enumAsInteger: true }));
		const id = generateRequestId(SimRequest.bulkSimAsync);

		signals.abort.onTrigger(async () => {
			await worker.sendAbortById(id);
		});

		const iterations = request.baseSettings?.simOptions?.iterations ?? 30000;
		const result = await this.doAsyncRequest(SimRequest.bulkSimAsync, BulkSimRequest.toBinary(request), id, worker, onProgress, iterations);

		const resultJson = BulkSimResult.toJson(result.finalBulkResult!) as any;
		worker.log('bulk sim result: ' + JSON.stringify(resultJson));
		return result.finalBulkResult!;
	}

	// Calculate combos and return counts
	async bulkSimCombosAsync(request: BulkSimCombosRequest): Promise<BulkSimCombosResult> {
		const worker = this.getLeastBusyWorker();
		worker.log('bulk sim combinations request: ' + BulkSimCombosRequest.toJsonString(request, { enumAsInteger: true }));
		const id = generateRequestId(SimRequest.bulkSimCombos);

		// Now start the async sim
		const resultData = await worker.doApiCall(SimRequest.bulkSimCombos, BulkSimCombosRequest.toBinary(request), id);
		return BulkSimCombosResult.fromBinary(resultData);
	}

	async raidSimAsync(request: RaidSimRequest, onProgress: WorkerProgressCallback, signals: SimSignals): Promise<RaidSimResult> {
		const worker = this.getLeastBusyWorker();
		worker.log('Raid sim request: ' + RaidSimRequest.toJsonString(request));
		const id = generateRequestId(SimRequest.raidSimAsync);

		signals.abort.onTrigger(async () => {
			await worker.sendAbortById(id);
		});

		const iterations = request.simOptions?.iterations ?? 3000;
		const result = await this.doAsyncRequest(SimRequest.raidSimAsync, RaidSimRequest.toBinary(request), id, worker, onProgress, iterations);

		// Don't print the logs because it just clogs the console.
		const resultJson = RaidSimResult.toJson(result.finalRaidResult!) as any;
		delete resultJson!['logs'];
		worker.log('Raid sim result: ' + JSON.stringify(resultJson));
		return result.finalRaidResult!;
	}

	async raidSimRequestSplit(request: RaidSimRequestSplitRequest): Promise<RaidSimRequestSplitResult> {
		const result = await this.makeApiCall(SimRequest.raidSimRequestSplit, RaidSimRequestSplitRequest.toBinary(request));
		return RaidSimRequestSplitResult.fromBinary(result);
	}

	async raidSimResultCombination(request: RaidSimResultCombinationRequest): Promise<RaidSimResult> {
		const result = await this.makeApiCall(SimRequest.raidSimResultCombination, RaidSimResultCombinationRequest.toBinary(request));
		return RaidSimResult.fromBinary(result);
	}

	/**
	 * Check if workers are net workers or wasm workers.
	 * @returns True if workers are running wasm.
	 */
	isWasm() {
		return this.isWasmPromise;
	}

	/**
	 * A promise that resolves when the first worker is ready.
	 * Just an alias for isWasm() tbh.
	 */
	waitForAnyWorkerReady() {
		return this.isWasmPromise;
	}

	/**
	 * Start an async request, handling progress and returning the final ProgressMetrics.
	 * @param requestName
	 * @param request
	 * @param id The task id used.
	 * @param worker
	 * @param onProgress
	 * @param totalIterations Used for initial work amount tracking for worker task.
	 * @returns The final ProgressMetrics.
	 */
	private async doAsyncRequest(
		requestName: SimRequest.raidSimAsync | SimRequest.bulkSimAsync | SimRequest.statWeightsAsync,
		request: Uint8Array,
		id: string,
		worker: SimWorker,
		onProgress: WorkerProgressCallback,
		totalIterations: number,
	): Promise<ProgressMetrics> {
		try {
			worker.addSimTaskRunning(id, totalIterations);
			worker.doApiCall(requestName, request, id);
			const finalProgress: Promise<ProgressMetrics> = new Promise((resolve, reject) => {
				// Add handler for the progress events
				worker.addPromiseFunc(this.getProgressName(id), this.newProgressHandler(id, worker, onProgress, resolve, reject), noop);
			});
			return await finalProgress;
		} finally {
			worker.updateSimTask(id, 0);
		}
	}

	private newProgressHandler(
		id: string,
		worker: SimWorker,
		onProgress: WorkerProgressCallback,
		onFinal: (pm: ProgressMetrics) => void,
		onError: (error: any) => void,
	): (progressData: Uint8Array) => void {
		return (progressData: any) => {
			const progress = ProgressMetrics.fromBinary(progressData);
			onProgress(progress);
			worker.updateSimTask(id, Math.max(1, progress.totalIterations - progress.completedIterations));
			// If we are done, stop adding the handler.
			if (progress.finalRaidResult != null || progress.finalWeightResult != null || progress.finalBulkResult != null) {
				onFinal(progress);
				return;
			}

			worker.addPromiseFunc(this.getProgressName(id), this.newProgressHandler(id, worker, onProgress, onFinal, onError), onError);
		};
	}
}

class SimWorker {
	readonly workerId: number;
	private readonly simTasksRunning: Record<string, { workLeft: number }>;
	private readonly taskIdsToPromiseFuncs: Record<string, [(result: any) => void, (error: any) => void]>;
	private readonly onStateChangeCallback: WorkerStateChangeCallback;
	private worker: Worker | undefined;
	private onReady: Promise<void> | undefined;
	private resolveReady: (() => void) | undefined;
	private rejectReady: ((error: any) => void) | undefined;
	private wasmWorker: boolean;
	private shouldDestroy: boolean;
	restartCount = 0;

	constructor(id: number, onStateChange: WorkerStateChangeCallback) {
		this.workerId = id;
		this.simTasksRunning = {};
		this.taskIdsToPromiseFuncs = {};
		this.wasmWorker = false;
		this.shouldDestroy = false;
		this.onStateChangeCallback = onStateChange;
		this.setupWorker();
		this.log('Created.');
	}

	private setupWorker() {
		this.setTaskActive('setup', true); // Make it prefer ready workers.

		this.onReady = new Promise((_resolve, _reject) => {
			this.resolveReady = _resolve;
			this.rejectReady = _reject;
		});

		this.worker = new window.Worker(SIM_WORKER_URL);

		this.worker.addEventListener('message', ({ data }: MessageEvent<WorkerSendMessage>) => {
			const { id, msg, outputData } = data;
			switch (msg) {
				case 'ready':
					this.wasmWorker = !!outputData && !!outputData[0];
					this.postMessage({ msg: 'setID', id: this.workerId.toString() });
					this.resolveReady!();
					this.setTaskActive('setup', false);
					this.onStateChangeCallback(this, 'ready');
					this.log(`Ready, isWasm: ${this.wasmWorker}`);
					break;
				case 'idConfirm':
					break;
				case 'workerError':
					this.rejectReady!(data.error);
					this.setTaskActive('setup', false);
					this.onStateChangeCallback(this, 'error', data.error);
					for (const taskId in this.simTasksRunning) {
						delete this.simTasksRunning[taskId];
					}
					for (const taskId in this.taskIdsToPromiseFuncs) {
						this.taskIdsToPromiseFuncs[taskId][1](data.error);
						delete this.taskIdsToPromiseFuncs[taskId];
					}
					break;
				default:
					const promiseFuncs = this.taskIdsToPromiseFuncs[id];
					if (!promiseFuncs) {
						console.warn(`Unrecognized result id ${id} for msg ${msg}`);
						return;
					}
					if (!id.includes('progress')) this.setTaskActive(id, false);
					delete this.taskIdsToPromiseFuncs[id];
					promiseFuncs[0](outputData);
			}
		});
	}

	/** Add sim work amount (iterations) used for load balancing. */
	addSimTaskRunning(id: string, workLeft: number) {
		this.simTasksRunning[id] = { workLeft };
		this.log(`Added work ${id}, current work amount: ${this.getSimTaskWorkAmount()}`);
	}

	/**
	 * Update sim work amount (iterations left) used for load balancing.
	 * @param id
	 * @param workLeft Set to 0 to remove the task.
	 */
	updateSimTask(id: string, workLeft: number) {
		if (workLeft <= 0) {
			delete this.simTasksRunning[id];
			this.log(`Work ${id} done, current work amount: ${this.getSimTaskWorkAmount()}`);
			if (this.shouldDestroy && this.getSimTaskWorkAmount() == 0) {
				this.disable(true);
			}
			return;
		}
		this.simTasksRunning[id].workLeft = workLeft;
	}

	/** Get total iterative work left on this worker. */
	getSimTaskWorkAmount() {
		let work = 0;
		for (const t of Object.values(this.simTasksRunning)) {
			work += t.workLeft;
		}
		return work;
	}

	/** Check if worker has a running task with id. */
	hasTaskId(id: string) {
		return !!this.simTasksRunning[id];
	}

	private setTaskActive(id: string, active: boolean) {
		if (active) {
			this.addSimTaskRunning(id + 'task', 1); // Add 1 work to track pending tasks
		} else {
			this.updateSimTask(id + 'task', 0);
		}
	}

	async isWasmWorker() {
		if (!this.onReady || this.shouldDestroy) throw new Error('Disabled worker was used!');
		await this.onReady;
		return this.wasmWorker;
	}

	addPromiseFunc(id: string, callback: (result: Uint8Array) => void, onError: (error: any) => void) {
		this.taskIdsToPromiseFuncs[id] = [callback, onError];
	}

	async doApiCall(requestName: SimRequest, request: Uint8Array, id: string): Promise<Uint8Array> {
		if (!this.onReady || this.shouldDestroy) throw new Error('Disabled worker was used!');
		if (!id) throw new Error('ApiCall with empty id!');
		this.setTaskActive(id, true);
		await this.onReady;

		const taskPromise = new Promise<Uint8Array>((resolve, reject) => {
			this.taskIdsToPromiseFuncs[id] = [resolve, reject];

			this.postMessage({
				msg: requestName,
				id,
				inputData: request,
			});
		});
		return await taskPromise;
	}

	postMessage(message: WorkerReceiveMessage) {
		if (!this.worker) throw new Error(`Worker ${this.workerId} postMessage while disabled!`);
		this.worker.postMessage(message);
	}

	disable(force = false) {
		this.shouldDestroy = true;
		if (!this.worker || (!force && this.getSimTaskWorkAmount())) return;
		for (const taskId in this.simTasksRunning) {
			delete this.simTasksRunning[taskId];
		}
		for (const taskId in this.taskIdsToPromiseFuncs) {
			delete this.taskIdsToPromiseFuncs[taskId];
		}
		this.onStateChangeCallback(this, 'disable');
		this.worker.terminate();
		delete this.worker;
		delete this.onReady;
		delete this.resolveReady;
		delete this.rejectReady;
		this.log('Disabled.');
	}

	enable() {
		this.shouldDestroy = false;
		if (this.worker) return;
		this.setupWorker();
		this.log('Enabled.');
	}

	log(s: string) {
		if (isDevMode()) console.log(`Worker ${this.workerId}: ${s}`);
	}

	async sendAbortById(requestId: string) {
		const abortReqBinary = AbortRequest.toBinary(AbortRequest.create({ requestId }));
		const rid = generateRequestId(SimRequest.abortById);
		const result = await this.doApiCall(SimRequest.abortById, abortReqBinary, rid);
		return AbortResponse.fromBinary(result);
	}
}
