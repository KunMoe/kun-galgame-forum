const isOpen = ref(false)
const targetPostId = ref<string | null>(null)

export const useGalgameCommentFlag = () => {
  const open = (postId: string) => {
    targetPostId.value = postId
    isOpen.value = true
  }
  return { isOpen, targetPostId, open }
}
