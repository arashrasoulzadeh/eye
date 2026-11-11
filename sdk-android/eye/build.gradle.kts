plugins {
    id("com.android.library")
    id("org.jetbrains.kotlin.android")
}

android { namespace = "ir.meetarash.eye"; compileSdk = 35
    defaultConfig { minSdk = 23; consumerProguardFiles("consumer-rules.pro") }
}

dependencies { testImplementation(kotlin("test")) }
