# GatherUp — Android Frontend (Full File Structure)

This document contains a complete, production-ready frontend file structure for the GatherUp Android app (Compose-first, Kotlin). It folds in recommended improvements (DI/Hilt, WS helpers, Paging 3, WorkManager, tests, CI, observability, feature modules) and shows **exact file paths** where to place files. Use the `/* Place: ... */` comments as TODO markers you can copy into files.

---

## Project root: `mobile/android/`

```
mobile/android/
├── app/                            # Main Android application module
│   ├── build.gradle.kts
│   ├── src/main/
│   │   ├── AndroidManifest.xml
│   │   ├── java/com/gatherup/
│   │   │   ├── MainActivity.kt
│   │   │   ├── App.kt
│   │   │   ├── di/
│   │   │   │   ├── HiltApp.kt                         /* Place: src/main/java/com/gatherup/di/HiltApp.kt */
│   │   │   │   ├── AppModule.kt
│   │   │   │   ├── NetworkModule.kt                   /* Place: src/main/java/com/gatherup/di/NetworkModule.kt */
│   │   │   │   ├── DatabaseModule.kt                  /* Place: src/main/java/com/gatherup/di/DatabaseModule.kt */
│   │   │   │   └── WsModule.kt                        /* Place: src/main/java/com/gatherup/di/WsModule.kt */
│   │   │   ├── security/
│   │   │   │   ├── SecurePrefs.kt
│   │   │   │   ├── BiometricAuth.kt
│   │   │   │   ├── CertificatePinning.kt
│   │   │   │   ├── TokenStore.kt                      /* Place: src/main/java/com/gatherup/security/TokenStore.kt */
│   │   │   │   └── DeviceIntegrityChecker.kt         /* Place: src/main/java/com/gatherup/security/DeviceIntegrityChecker.kt */
│   │   │   ├── observability/
│   │   │   │   ├── CrashReporter.kt                  /* Place: src/main/java/com/gatherup/observability/CrashReporter.kt */
│   │   │   │   └── Analytics.kt                      /* Place: src/main/java/com/gatherup/observability/Analytics.kt */
│   │   │   ├── ui/
│   │   │   │   ├── theme/
│   │   │   │   │   ├── Color.kt
│   │   │   │   │   ├── Typography.kt
│   │   │   │   │   ├── Theme.kt
│   │   │   │   │   └── designTokens/                 /* Place: src/main/java/com/gatherup/ui/theme/designtokens/ */
│   │   │   │   ├── navigation/
│   │   │   │   │   ├── NavGraph.kt
│   │   │   │   │   ├── Destinations.kt
│   │   │   │   │   └── NavigationViewModel.kt
│   │   │   │   ├── screens/
│   │   │   │   │   ├── auth/
│   │   │   │   │   │   ├── LoginScreen.kt
│   │   │   │   │   │   ├── RegisterScreen.kt
│   │   │   │   │   │   └── ForgotPasswordScreen.kt
│   │   │   │   │   ├── FeedScreen.kt
│   │   │   │   │   ├── PostDetailScreen.kt
│   │   │   │   │   ├── PostComposerScreen.kt
│   │   │   │   │   ├── ChatListScreen.kt
│   │   │   │   │   ├── ChatThreadScreen.kt
│   │   │   │   │   ├── TournamentListScreen.kt
│   │   │   │   │   ├── TournamentDetailScreen.kt
│   │   │   │   │   ├── CreateTournamentScreen.kt
│   │   │   │   │   ├── ProfileScreen.kt
│   │   │   │   │   └── SettingsScreen.kt
│   │   │   │   └── components/
│   │   │   │       ├── common/
│   │   │   │       │   ├── Avatar.kt
│   │   │   │       │   ├── LoadingIndicator.kt
│   │   │   │       │   ├── ErrorState.kt
│   │   │   │       │   ├── SearchBar.kt
│   │   │   │       │   └── BottomNavBar.kt
│   │   │   │       ├── posts/
│   │   │   │       │   ├── PostCard.kt
│   │   │   │       │   ├── PostActions.kt
│   │   │   │       │   ├── CommentItem.kt
│   │   │   │       │   └── MediaGallery.kt
│   │   │   │       ├── chat/
│   │   │   │       │   ├── MessageBubble.kt
│   │   │   │       │   ├── TypingIndicator.kt
│   │   │   │       │   ├── ChatHeader.kt
│   │   │   │       │   └── MessageComposer.kt
│   │   │   │       └── tournaments/
│   │   │   │           ├── TournamentCard.kt
│   │   │   │           ├── LeaderboardItem.kt
│   │   │   │           ├── MatchCard.kt
│   │   │   │           └── ParticipantList.kt
│   │   │   ├── ws/
│   │   │   │   ├── WsClient.kt
│   │   │   │   ├── WsModels.kt
│   │   │   │   ├── WsMessageHandler.kt
│   │   │   │   ├── WsReconnectionManager.kt
│   │   │   │   ├── WsHeartbeatManager.kt              /* Place: src/main/java/com/gatherup/ws/WsHeartbeatManager.kt */
│   │   │   │   ├── WsStateFlow.kt                     /* Place: src/main/java/com/gatherup/ws/WsStateFlow.kt */
│   │   │   │   └── serializers/
│   │   │   │       └── WsJsonAdapters.kt              /* Place: src/main/java/com/gatherup/ws/serializers/WsJsonAdapters.kt */
│   │   │   ├── data/
│   │   │   │   ├── api/
│   │   │   │   │   ├── ApiService.kt
│   │   │   │   │   ├── AuthService.kt
│   │   │   │   │   ├── PostService.kt
│   │   │   │   │   ├── ChatService.kt
│   │   │   │   │   ├── TournamentService.kt
│   │   │   │   │   ├── UserService.kt
│   │   │   │   │   ├── interceptors/
│   │   │   │   │   │   ├── AuthInterceptor.kt
│   │   │   │   │   │   ├── LoggingInterceptor.kt
│   │   │   │   │   │   └── RetryInterceptor.kt
│   │   │   │   │   └── adapters/
│   │   │   │   │       ├── DateTimeAdapter.kt
│   │   │   │   │       └── UUIDAdapter.kt
│   │   │   │   ├── paging/
│   │   │   │   │   ├── PostRemoteMediator.kt         /* Place: src/main/java/com/gatherup/data/paging/PostRemoteMediator.kt */
│   │   │   │   │   └── MessageRemoteMediator.kt      /* Place: src/main/java/com/gatherup/data/paging/MessageRemoteMediator.kt */
│   │   │   │   ├── local/
│   │   │   │   │   ├── GatherUpDatabase.kt
│   │   │   │   │   ├── dao/
│   │   │   │   │   │   ├── UserDao.kt
│   │   │   │   │   │   ├── PostDao.kt
│   │   │   │   │   │   ├── MessageDao.kt
│   │   │   │   │   │   └── TournamentDao.kt
│   │   │   │   │   └── migrations/                   /* Place: src/main/java/com/gatherup/local/migrations/ */
│   │   │   │   ├── repository/
│   │   │   │   │   ├── UserRepositoryImpl.kt
│   │   │   │   │   ├── PostRepositoryImpl.kt
│   │   │   │   │   ├── ChatRepositoryImpl.kt
│   │   │   │   │   ├── TournamentRepositoryImpl.kt
│   │   │   │   │   └── AuthRepositoryImpl.kt
│   │   │   │   └── models/
│   │   │   │       ├── User.kt
│   │   │   │       ├── Post.kt
│   │   │   │       ├── Message.kt
│   │   │   │       ├── Tournament.kt
│   │   │   │       ├── ApiResponse.kt
│   │   │   │       └── WsMessage.kt
│   │   │   ├── domain/
│   │   │   │   ├── repository/
│   │   │   │   │   ├── UserRepository.kt
│   │   │   │   │   ├── PostRepository.kt
│   │   │   │   │   ├── ChatRepository.kt
│   │   │   │   │   └── TournamentRepository.kt
│   │   │   │   ├── models/
│   │   │   │   │   ├── User.kt
│   │   │   │   │   ├── Post.kt
│   │   │   │   │   ├── Message.kt
│   │   │   │   │   └── Tournament.kt
│   │   │   │   └── usecase/
│   │   │   │       ├── auth/
│   │   │   │       │   ├── LoginUseCase.kt
│   │   │   │       │   ├── RegisterUseCase.kt
│   │   │   │       │   └── LogoutUseCase.kt
│   │   │   │       ├── posts/
│   │   │   │       │   ├── GetFeedUseCase.kt
│   │   │   │       │   ├── CreatePostUseCase.kt
│   │   │   │       │   ├── ReactToPostUseCase.kt
│   │   │   │       │   └── CommentOnPostUseCase.kt
│   │   │   │       ├── chat/
│   │   │   │       │   ├── SendMessageUseCase.kt
│   │   │   │       │   ├── GetChatHistoryUseCase.kt
│   │   │   │       │   ├── MarkAsReadUseCase.kt
│   │   │   │       │   └── CreateGroupChatUseCase.kt
│   │   │   │       └── tournaments/
│   │   │   │           ├── JoinTournamentUseCase.kt
│   │   │   │           ├── GetLeaderboardUseCase.kt
│   │   │   │           ├── CreateTournamentUseCase.kt
│   │   │   │           └── UpdateMatchResultUseCase.kt
│   │   │   ├── vm/
│   │   │   │   ├── AuthViewModel.kt
│   │   │   │   ├── FeedViewModel.kt
│   │   │   │   ├── ChatViewModel.kt
│   │   │   │   ├── TournamentViewModel.kt
│   │   │   │   ├── ProfileViewModel.kt
│   │   │   │   └── SettingsViewModel.kt
│   │   │   └── util/
│   │   │       ├── Extensions.kt
│   │   │       ├── Result.kt
│   │   │       ├── DateTimeUtils.kt
│   │   │       ├── ImageUtils.kt
│   │   │       ├── NetworkUtils.kt
│   │   │       ├── PermissionManager.kt
│   │   │       ├── LocationHelper.kt
│   │   │       └── Validators.kt
│   │   ├── res/
│   │   │   ├── drawable/
│   │   │   ├── layout/
│   │   │   └── values/
│   │   ├── test/
│   │   │   └── java/com/gatherup/
│   │   │       ├── domain/usecase/
│   │   │       ├── vm/
│   │   │       └── repository/
│   │   └── androidTest/
│   │       └── java/com/gatherup/
│   │           ├── ui/screens/
│   │           └── repository/
│   └── proguard-rules.pro
├── features/                        # Optional: feature modules to be converted to Gradle modules later
│   ├── feed/
│   ├── chat/
│   └── tournaments/
├── gradle/
│   ├── dependencies.gradle.kts
│   └── config.gradle.kts
├── scripts/
│   └── seed_emulator_data.sh
├── .editorconfig
├── gradle.properties
├── detekt.yml                        /* Place: detekt.yml */
├── .github/
│   └── workflows/
│       ├── android-ci.yml            /* Place: .github/workflows/android-ci.yml */
│       └── lint.yml                  /* Place: .github/workflows/lint.yml */
└── README.md
```

---

## Explanation of added folders/files (short)

* `di/` — Hilt wiring and modules (AppModule, NetworkModule, DatabaseModule, WsModule). HiltApp annotates the Application class.
* `ws/` — WebSocket client, reconnection manager, heartbeat, and StateFlow to expose connection status to ViewModels/Services.
* `data/paging/` — RemoteMediator implementations for Paging 3 to support offline + pagination.
* `local/migrations/` — Room migration scripts. Important to add for DB schema changes.
* `security/TokenStore.kt` — Encrypted token storage with Android Keystore/EncryptedSharedPreferences.
* `observability/` — Crash reporter & analytics facade.
* `features/` — Suggested folder to split into dynamic feature modules later for faster builds.
* `.github/workflows/` + `detekt.yml` — CI pipeline plus linting/static analysis.

---

## Small code snippets (place these files)

`/* src/main/java/com/gatherup/ws/WsConnectionState.kt */`

```kotlin
package com.gatherup.ws

enum class WsConnectionState { CONNECTING, OPEN, BACKOFF, CLOSED }
```

`/* src/main/java/com/gatherup/data/api/NetworkResult.kt */`

```kotlin
package com.gatherup.data.api

sealed class NetworkResult<out T> {
  data class Success<T>(val data: T): NetworkResult<T>()
  data class Failure(val error: ApiError): NetworkResult<Nothing>()
}
```

---

## Next steps & recommended priorities (recap)

1. Add DI (Hilt) + NetworkModule.
2. Implement WS heartbeat + reconnection + StateFlow.
3. Add Paging 3 + RemoteMediator for feed & chat.
4. Implement TokenStore (EncryptedSharedPreferences).
5. Add WorkManager flows for uploads/sync.
6. Add tests (unit + Compose UI + MockWebServer).
7. Configure CI (GitHub Actions) + detekt/ktlint.

---

If you want, I can now:

* generate a ready-to-use `NetworkModule.kt` (Hilt + Retrofit + OkHttp + WebSocket wiring), or
* scaffold `PostRemoteMediator.kt` for Paging 3, or
* produce `android-ci.yml` for GitHub Actions.

Pick one and I will generate the complete file with code you can paste into `mobile/android/app/src/main/...`.
