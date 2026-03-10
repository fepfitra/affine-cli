package graphql

// ── Workspace ────────────────────────────────────────────────────────

const ListWorkspacesQuery = `query { workspaces { id public enableAi createdAt } }`

const GetWorkspaceQuery = `query GetWorkspace($id: String!) {
  workspace(id: $id) {
    id public enableAi createdAt
    permissions { Workspace_Read Workspace_CreateDoc }
  }
}`

const CreateWorkspaceMutation = `mutation createWorkspace($init: Upload!) {
  createWorkspace(init: $init) { id public createdAt enableAi }
}`

const UpdateWorkspaceMutation = `mutation UpdateWorkspace($input: UpdateWorkspaceInput!) {
  updateWorkspace(input: $input) { id public enableAi }
}`

const DeleteWorkspaceMutation = `mutation DeleteWorkspace($id: String!) {
  deleteWorkspace(id: $id)
}`

// ── Docs ─────────────────────────────────────────────────────────────

const ListDocsQuery = `query ListDocs($workspaceId:String!,$first:Int,$offset:Int,$after:String){
  workspace(id:$workspaceId){
    docs(pagination:{first:$first,offset:$offset,after:$after}){
      totalCount
      pageInfo{ hasNextPage endCursor }
      edges{ cursor node{ id workspaceId title summary public defaultRole createdAt updatedAt } }
    }
  }
}`

const GetDocQuery = `query GetDoc($workspaceId:String!,$docId:String!){
  workspace(id:$workspaceId){
    doc(docId:$docId){ id workspaceId title summary public defaultRole createdAt updatedAt }
  }
}`

const PublishDocMutation = `mutation PublishDoc($workspaceId:String!,$docId:String!,$mode:PublicDocMode){
  publishDoc(workspaceId:$workspaceId,docId:$docId,mode:$mode){ id workspaceId public mode }
}`

const RevokeDocMutation = `mutation RevokeDoc($workspaceId:String!,$docId:String!){
  revokePublicDoc(workspaceId:$workspaceId,docId:$docId){ id workspaceId public }
}`

// ── Comments ─────────────────────────────────────────────────────────

const ListCommentsQuery = `query ListComments($workspaceId:String!,$docId:String!,$first:Int,$offset:Int,$after:String){
  workspace(id:$workspaceId){
    comments(docId:$docId,pagination:{first:$first,offset:$offset,after:$after}){
      totalCount
      pageInfo{ hasNextPage endCursor }
      edges{ cursor node{
        id content createdAt updatedAt resolved
        user{ id name avatarUrl }
        replies{ id content createdAt updatedAt user{ id name avatarUrl } }
      }}
    }
  }
}`

const CreateCommentMutation = `mutation CreateComment($input:CommentCreateInput!){
  createComment(input:$input){ id content createdAt updatedAt resolved }
}`

const UpdateCommentMutation = `mutation UpdateComment($input:CommentUpdateInput!){
  updateComment(input:$input)
}`

const DeleteCommentMutation = `mutation DeleteComment($id:String!){
  deleteComment(id:$id)
}`

const ResolveCommentMutation = `mutation ResolveComment($input:CommentResolveInput!){
  resolveComment(input:$input)
}`

// ── History ──────────────────────────────────────────────────────────

const ListHistoriesQuery = `query Histories($workspaceId:String!,$guid:String!,$take:Int,$before:DateTime){
  workspace(id:$workspaceId){
    histories(guid:$guid,take:$take,before:$before){ id timestamp workspaceId }
  }
}`

// ── Notifications ────────────────────────────────────────────────────

const ListNotificationsQuery = `query GetNotifications($pagination:PaginationInput!){
  currentUser{
    notifications(pagination:$pagination){
      totalCount
      pageInfo{ hasNextPage endCursor }
      edges{ cursor node{ id type body read level createdAt updatedAt } }
    }
  }
}`

const ReadAllNotificationsMutation = `mutation ReadAllNotifications { readAllNotifications }`

// ── User ─────────────────────────────────────────────────────────────

const CurrentUserQuery = `query Me { currentUser { id name email emailVerified avatarUrl disabled } }`

const UpdateProfileMutation = `mutation UpdateProfile($input:UpdateUserInput!){
  updateProfile(input:$input){ id name avatarUrl email }
}`

const UpdateSettingsMutation = `mutation UpdateSettings($input:UpdateUserSettingsInput!){
  updateSettings(input:$input)
}`

// ── Access Tokens ────────────────────────────────────────────────────

const ListAccessTokensQuery = `query { currentUser { accessTokens { id name createdAt expiresAt } } }`

const GenerateAccessTokenMutation = `mutation($input:GenerateAccessTokenInput!){
  generateUserAccessToken(input:$input){ id name createdAt expiresAt token }
}`

const RevokeAccessTokenMutation = `mutation($id:String!){ revokeUserAccessToken(id:$id) }`

// ── Blob Storage ─────────────────────────────────────────────────────

const SetBlobMutation = `mutation SetBlob($workspaceId:String!,$blob:Upload!){
  setBlob(workspaceId:$workspaceId,blob:$blob)
}`

const DeleteBlobMutation = `mutation DeleteBlob($workspaceId:String!,$key:String!,$permanently:Boolean){
  deleteBlob(workspaceId:$workspaceId,key:$key,permanently:$permanently)
}`

const CleanupBlobsMutation = `mutation ReleaseDeletedBlobs($workspaceId:String!){
  releaseDeletedBlobs(workspaceId:$workspaceId)
}`
