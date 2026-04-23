import { useRef, useMemo } from 'react';
import { Canvas, useFrame } from '@react-three/fiber';
import * as THREE from 'three';

const STAGE_COLORS: Record<number, string> = {
  1: '#a5b4fc',
  2: '#fbbf24',
  3: '#34d399',
  4: '#c084fc',
};

function ParticleOrb({ stage, size = 1 }: { stage: number; size?: number }) {
  const meshRef = useRef<THREE.Points>(null);
  const color = STAGE_COLORS[stage] ?? STAGE_COLORS[1];
  const count = 600 + stage * 200;

  const positions = useMemo(() => {
    const pos = new Float32Array(count * 3);
    for (let i = 0; i < count; i++) {
      const theta = Math.random() * Math.PI * 2;
      const phi = Math.acos(2 * Math.random() - 1);
      const r = (0.6 + Math.random() * 0.4) * size;
      pos[i * 3] = r * Math.sin(phi) * Math.cos(theta);
      pos[i * 3 + 1] = r * Math.sin(phi) * Math.sin(theta);
      pos[i * 3 + 2] = r * Math.cos(phi);
    }
    return pos;
  }, [count, size]);

  useFrame((state) => {
    if (!meshRef.current) return;
    const t = state.clock.elapsedTime;
    meshRef.current.rotation.y = t * 0.15;
    meshRef.current.rotation.x = Math.sin(t * 0.1) * 0.1;

    // Breathing scale
    const breathe = 1 + Math.sin(t * 1.5) * 0.04;
    meshRef.current.scale.setScalar(breathe);
  });

  return (
    <points ref={meshRef}>
      <bufferGeometry>
        <bufferAttribute
          attach="attributes-position"
          args={[positions, 3]}
        />
      </bufferGeometry>
      <pointsMaterial
        size={0.025 * size}
        color={color}
        transparent
        opacity={0.8}
        sizeAttenuation
        blending={THREE.AdditiveBlending}
        depthWrite={false}
      />
    </points>
  );
}

function GlowCore({ stage, size = 1 }: { stage: number; size?: number }) {
  const meshRef = useRef<THREE.Mesh>(null);
  const color = STAGE_COLORS[stage] ?? STAGE_COLORS[1];

  useFrame((state) => {
    if (!meshRef.current) return;
    const t = state.clock.elapsedTime;
    const s = (0.3 + stage * 0.05) * size;
    const breathe = s + Math.sin(t * 2) * 0.02 * size;
    meshRef.current.scale.setScalar(breathe);
  });

  return (
    <mesh ref={meshRef}>
      <sphereGeometry args={[1, 32, 32]} />
      <meshBasicMaterial color={color} transparent opacity={0.15} />
    </mesh>
  );
}

export default function SpiritOrb({
  stage,
  size = 'md',
}: {
  stage: number;
  size?: 'sm' | 'md' | 'lg' | 'full';
}) {
  if (size === 'full') {
    return (
      <div style={{ width: '100%', height: '100%', position: 'absolute', inset: 0 }}>
        <Canvas
          camera={{ position: [0, 0, 3], fov: 60 }}
          gl={{ alpha: true, antialias: true }}
          style={{ background: 'transparent' }}
        >
          <ambientLight intensity={0.5} />
          <ParticleOrb stage={stage} size={3.5} />
          <GlowCore stage={stage} size={3.5} />
        </Canvas>
      </div>
    );
  }

  const dims = { sm: 48, md: 120, lg: 240 }[size];
  const scale = { sm: 0.6, md: 1, lg: 1.8 }[size];

  return (
    <div style={{ width: dims, height: dims }} className="flex-shrink-0">
      <Canvas
        camera={{ position: [0, 0, 2.5], fov: 50 }}
        gl={{ alpha: true, antialias: true }}
        style={{ background: 'transparent' }}
      >
        <ambientLight intensity={0.5} />
        <ParticleOrb stage={stage} size={scale} />
        <GlowCore stage={stage} size={scale} />
      </Canvas>
    </div>
  );
}
