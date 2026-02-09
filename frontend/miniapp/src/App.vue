<script lang="ts">
import { applyNavigationBarTheme, applyTabBarTheme, bootstrapTheme } from './utils/theme'

export default {
  onLaunch() {
    bootstrapTheme()
  },
  onShow() {
    applyNavigationBarTheme()
    applyTabBarTheme()
  },
}
</script>

<style>
page {
  --brand-1: #0d0d0d;
  --brand-2: #333333;
  --brand-strong: #1f1f1f;
  --brand-solid: #111111;
  --brand-soft: #d3d3d3;
  background: #f6f7fb;
  color: #111;
  font-size: 28rpx;
}
.confirm-btn {
  height: 72rpx;
  line-height: 72rpx;
  font-size: 28rpx;
  font-weight: 600;
  border-radius: 14rpx;
  background: var(--confirm-btn-bg-rgba, rgba(241, 241, 244, 0.9));
  color: var(--confirm-btn-color, #111111);
}
.confirm-btn::after {
  border: none;
}
</style>
