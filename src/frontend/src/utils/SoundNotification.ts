/**
 * SoundNotification - 分析完成声音提示
 * 
 * 使用 Web Audio API 生成提示音，无需外部音频文件
 */

let audioContext: AudioContext | null = null;

function getAudioContext(): AudioContext {
  if (!audioContext) {
    audioContext = new AudioContext();
  }
  return audioContext;
}

/**
 * 播放分析完成提示音
 * 两个短促的上升音调，清脆友好
 */
export function playAnalysisCompleteSound(): void {
  try {
    const ctx = getAudioContext();

    // 如果 AudioContext 被浏览器挂起，尝试恢复
    if (ctx.state === 'suspended') {
      ctx.resume();
    }

    const now = ctx.currentTime;

    // 第一个音符 - C5 (523Hz)
    playTone(ctx, 523, now, 0.12);
    // 第二个音符 - E5 (659Hz)，稍高一点表示"完成"
    playTone(ctx, 659, now + 0.15, 0.12);
  } catch (e) {
    // 静默失败，声音提示不应影响主流程
    console.warn('[SoundNotification] Failed to play sound:', e);
  }
}

function playTone(ctx: AudioContext, frequency: number, startTime: number, duration: number): void {
  const oscillator = ctx.createOscillator();
  const gainNode = ctx.createGain();

  oscillator.type = 'sine';
  oscillator.frequency.setValueAtTime(frequency, startTime);

  // 音量包络：快速淡入，平滑淡出
  gainNode.gain.setValueAtTime(0, startTime);
  gainNode.gain.linearRampToValueAtTime(0.3, startTime + 0.02);
  gainNode.gain.exponentialRampToValueAtTime(0.01, startTime + duration);

  oscillator.connect(gainNode);
  gainNode.connect(ctx.destination);

  oscillator.start(startTime);
  oscillator.stop(startTime + duration);
}
