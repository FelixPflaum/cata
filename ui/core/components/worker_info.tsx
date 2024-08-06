import tippy from 'tippy.js';
import { ref } from 'tsx-vanilla';

import { Sim } from '../sim';

/**
 * @param parent If not null will append itself to it.
 * @param sim
 * @param hideParent If true will hide the parent instead of itself when not running wasm.
 */
export const buildWorkerInfo = (parent: JSX.Element | null, sim: Sim, hideParent = false): Element => {
	const textSpanRef = ref<HTMLSpanElement>();
	const iconRef = ref<HTMLElement>();
	const content = (
		<span className="worker-info">
			<i ref={iconRef} className="fas fa-spinner fa-spin" />
			<span ref={textSpanRef}> Workers loading...</span>
		</span>
	);
	const toolTip = tippy(parent ?? content, {
		content: 'WebWorker status.',
		placement: 'bottom',
	});

	if (parent) parent.appendChild(content);

	sim.isWasm().then(isWasm => {
		if (!isWasm) {
			if (hideParent && parent) {
				parent.style.display = 'none';
			} else {
				content.style.display = 'none';
			}
		}
	});

	sim.workerNumChangeEmitter.on((_, nums) => {
		let status = ` ${nums.ready}/${nums.numSet} Workers running`;
		if (nums.numSet == nums.ready) {
			iconRef.value!.style.display = 'none';
			toolTip.setContent('All workers running.');
		} else {
			iconRef.value!.className = 'fas fa-exclamation-triangle link-warning';
			iconRef.value!.style.display = '';
			if (nums.numSet != nums.enabled) {
				toolTip.setContent('Some workers failed to load! Number of workers was reduced. The browser console may have more details.');
				status += ` (${nums.numSet - nums.enabled} failed!)`
			} else {
				toolTip.setContent('Workers are still loading!');
			}
		}
		textSpanRef.value!.innerText = status;
	});

	return content;
};
