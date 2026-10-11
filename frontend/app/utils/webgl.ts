// WebGL非対応・低スペ判定
export function canUseWebGL(): boolean {
	try {
		const canvas = window.document.createElement('canvas');
		const gl = canvas.getContext('webgl2') ?? canvas.getContext('webgl');
		return gl !== null && navigator.hardwareConcurrency >= 4;
	} catch {
		return false;
	}
}
