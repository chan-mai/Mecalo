import * as THREE from 'three';

export const outlineColor = 0x25324a;

// 3段階の陰影
export function toonGradient(): THREE.DataTexture {
	const tex = new THREE.DataTexture(new Uint8Array([110, 190, 255]), 3, 1, THREE.RedFormat);
	tex.minFilter = THREE.NearestFilter;
	tex.magFilter = THREE.NearestFilter;
	tex.needsUpdate = true;
	return tex;
}

// 裏面を拡大して描く輪郭線
export function outlineOf(mesh: THREE.Mesh, scale: number, color: THREE.ColorRepresentation = outlineColor): THREE.Mesh {
	const line = new THREE.Mesh(mesh.geometry, new THREE.MeshBasicMaterial({ color, side: THREE.BackSide }));
	line.position.copy(mesh.position);
	line.quaternion.copy(mesh.quaternion);
	line.scale.copy(mesh.scale).multiplyScalar(scale);
	return line;
}

interface ToonOptions {
	outlineScale?: number;
	outline?: THREE.ColorRepresentation;
	// 白へ寄せる割合
	pale?: number;
}

// 元の色を保ったトゥーン材質と輪郭線へ置換
export function toonify(root: THREE.Object3D, gradientMap: THREE.Texture, { outlineScale = 1.04, outline = outlineColor, pale = 0 }: ToonOptions = {}): void {
	const meshes: THREE.Mesh[] = [];
	root.traverse((o) => {
		if (o instanceof THREE.Mesh) meshes.push(o);
	});
	for (const mesh of meshes) {
		const source = Array.isArray(mesh.material) ? mesh.material[0] : mesh.material;
		const base = source && 'color' in source ? (source.color as THREE.Color).clone() : new THREE.Color(0xffffff);
		mesh.material = new THREE.MeshToonMaterial({ color: base.lerp(new THREE.Color(0xffffff), pale), gradientMap });
		mesh.parent?.add(outlineOf(mesh, outlineScale, outline));
	}
}
