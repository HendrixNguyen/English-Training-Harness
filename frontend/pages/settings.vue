<template>
  <main class="mx-auto max-w-md px-4 pb-8">
    <AppHeader />
    <AppCard title="Sắp ra mắt" class="mt-2">
      <p class="text-mute">
        Nhắc học và đồng bộ Google sẽ bật ở đây khi máy chủ sẵn sàng.
      </p>
      <div class="mt-4 flex flex-col gap-2">
        <AppButton block disabled aria-describedby="settings-soon">Bật nhắc học</AppButton>
        <AppButton block disabled variant="ghost" aria-describedby="settings-soon">Đồng bộ Google</AppButton>
      </div>
      <p id="settings-soon" class="sr-only">Chưa khả dụng</p>
    </AppCard>
  </main>
</template>
